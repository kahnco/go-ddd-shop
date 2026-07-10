package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/ordering/api"
	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// main 은 "조립 루트(composition root)".
// 여기서만 구체 어댑터를 골라 포트에 끼운다. 도메인·애플리케이션 코드는
// 이 파일이 무엇을 고르는지 전혀 모른다 — 그래서 저장소를 인메모리에서 PostgreSQL 로,
// 발행을 로그에서 NATS 로 바꿔도 이 파일 몇 줄만 바뀐다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	ids := infra.RandomIDGenerator{}

	// 저장소 선택. DATABASE_URL 이 있으면 PostgreSQL(여러 파드가 상태를 공유),
	// 없으면 인메모리(단독 실행). readiness probe 는 저장소 종류에 맞게 준비된다.
	var repo domain.OrderRepository = infra.NewMemoryOrderRepository()
	var pg *infra.PostgresOrderRepository               // 아웃박스 릴레이가 참조하려고 따로 잡아 둔다
	ready := func(context.Context) error { return nil } // 인메모리는 항상 준비됨
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		var err error
		pg, err = infra.NewPostgresOrderRepository(ctx, dsn)
		if err != nil {
			logger.Error("postgres 연결 실패", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		repo = pg
		ready = pg.Ping // readiness = DB 가 응답하는가
		logger.Info("주문 저장소 = PostgreSQL")
	}

	var bus *eventbus.Bus
	if url := os.Getenv("NATS_URL"); url != "" {
		var err error
		bus, err = eventbus.Connect(url, eventbus.OptionsFromEnv()...)
		if err != nil {
			logger.Error("nats 연결 실패", "url", url, "err", err)
			os.Exit(1)
		}
		defer bus.Close()
	}

	// 발행 방식 선택.
	//   - Postgres + NATS → 트랜잭셔널 아웃박스: 이벤트는 저장 트랜잭션으로 아웃박스에 적재되고
	//     릴레이가 발행한다. 유스케이스의 직접 발행은 no-op 으로 끈다(이중 발행 방지).
	//   - NATS 만 → 직접 발행. DB 가 없으니 아웃박스도 의미 없다.
	//   - 둘 다 없음 → 로그 발행(단독 실행).
	var publisher app.EventPublisher = infra.NewLogPublisher(logger)
	switch {
	case pg != nil && bus != nil:
		publisher = infra.NoopPublisher{}
		relay := infra.NewOutboxRelay(pg, bus, 200*time.Millisecond, logger)
		go relay.Run(ctx)
		logger.Info("이벤트 발행 = 트랜잭셔널 아웃박스 + 릴레이")
	case bus != nil:
		publisher = infra.NewNatsEventPublisher(bus, "ordering")
		logger.Info("이벤트 발행 = NATS(직접)")
	}

	// 가격 프로젝션(읽기 모델). 카탈로그가 가격의 소유자이고, 주문은 이 사본에서 가격을 읽는다.
	projection := infra.NewProductProjection()
	if catalogURL := os.Getenv("CATALOG_URL"); catalogURL != "" {
		if err := projection.Bootstrap(ctx, catalogURL); err != nil {
			logger.Warn("카탈로그 부트스트랩 실패(이벤트로 채워질 수 있음)", "err", err)
		} else {
			logger.Info("카탈로그 부트스트랩 완료", "url", catalogURL)
		}
	}
	if projection.Empty() {
		// 카탈로그 없이 단독 실행할 때의 기본 상품.
		projection.SeedDefault("prod-A", 1000)
		projection.SeedDefault("prod-B", 3000)
	}

	svc := app.NewOrderService(repo, publisher, ids, projection)

	// 사가 구독: 결제 완료 → 주문 확정, 재고 부족 → 주문 취소.
	// 주문 서비스가 다른 컨텍스트의 이벤트에 반응해 주문 여정을 이어간다.
	if bus != nil {
		saga := infra.NewOrderSagaConsumer(svc, logger)
		subs := map[string]eventbus.Handler{
			"payment.completed":                  saga.OnPaymentCompleted,       // 결제 완료 → 확정
			"payment.failed":                     saga.OnPaymentFailed,          // 결제 실패 → 취소
			"shipping.dispatched":                saga.OnShipmentDispatched,     // 배송 시작 → 배송중
			"inventory.stock.reservation_failed": saga.OnStockReservationFailed, // 재고 부족 → 취소
			"catalog.product.added":              projection.OnProductAdded,     // 상품 등록 → 가격 프로젝션 갱신
			"catalog.product.price_changed":      projection.OnProductPriceChanged,
		}
		for subject, handler := range subs {
			if err := bus.Subscribe(subject, "ordering", handler); err != nil {
				logger.Error("구독 실패", "subject", subject, "err", err)
				os.Exit(1)
			}
		}
		logger.Info("구독 시작 — 사가(payment·shipping·stock) + 카탈로그(product) 이벤트")
	}

	mux := http.NewServeMux()
	api.NewOrderHandler(svc).Register(mux)
	api.RegisterHealth(mux, ready)
	mux.Handle("GET /metrics", telemetry.MetricsHandler()) // 프로메테우스 스크레이프 대상

	// 미들웨어로 감싸 상관 ID·접근 로그·HTTP 메트릭을 모든 요청에 적용한다.
	handler := telemetry.Middleware(logger, mux)

	addr := ":8080"
	logger.Info("ordering service 시작", "addr", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		logger.Error("서버 종료", "err", err)
		os.Exit(1)
	}
}
