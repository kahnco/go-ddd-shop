package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

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
	ready := func(context.Context) error { return nil } // 인메모리는 항상 준비됨
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pg, err := infra.NewPostgresOrderRepository(ctx, dsn)
		if err != nil {
			logger.Error("postgres 연결 실패", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		repo = pg
		ready = pg.Ping // readiness = DB 가 응답하는가
		logger.Info("주문 저장소 = PostgreSQL")
	}

	// 발행 어댑터 선택. NATS_URL 이 있으면 브로커로, 없으면 로그로.
	var publisher app.EventPublisher = infra.NewLogPublisher(logger)
	if url := os.Getenv("NATS_URL"); url != "" {
		bus, err := eventbus.Connect(url)
		if err != nil {
			logger.Error("nats 연결 실패", "url", url, "err", err)
			os.Exit(1)
		}
		defer bus.Close()
		publisher = infra.NewNatsEventPublisher(bus, "ordering")
		logger.Info("이벤트 발행 = NATS", "url", url)
	}

	svc := app.NewOrderService(repo, publisher, ids)

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
