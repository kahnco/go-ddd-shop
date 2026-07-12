package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
	"github.com/kahnco/go-ddd-shop/internal/inventory/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// inventory 서비스: 주문 컨텍스트의 order.placed 를 구독해 재고를 예약하는 소비자.
// 비즈니스 입구는 오직 이벤트지만, 관찰성(/metrics)과 probe(/healthz)를 위해
// 최소한의 HTTP 서버는 연다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shutdown, _ := telemetry.InitTracer(context.Background(), "inventory")
	defer func() { _ = shutdown(context.Background()) }()

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	// 재고 저장소 선택. DATABASE_URL 이 있으면 PostgreSQL(행 잠금으로 동시성 안전 + 여러
	// replica 가 재고를 공유), 없으면 인메모리(원자 Update 로 단일 인스턴스 동시성은 안전).
	var stockRepo domain.StockRepository
	var reservations domain.ReservationRepository
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pg, err := infra.NewPostgresStore(context.Background(), dsn)
		if err != nil {
			logger.Error("postgres 연결 실패", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		_ = pg.Seed(context.Background(), "prod-A", 10)
		_ = pg.Seed(context.Background(), "prod-B", 5)
		stockRepo, reservations = pg, pg
		logger.Info("재고 저장소 = PostgreSQL(다중 replica 안전)")
	} else {
		mem := infra.NewMemoryStockRepository()
		mem.Seed("prod-A", 10)
		mem.Seed("prod-B", 5)
		stockRepo = mem
		reservations = infra.NewMemoryReservationRepository()
	}

	publisher := infra.NewNatsEventPublisher(bus, "inventory")
	svc := app.NewReservationService(stockRepo, reservations, publisher)

	placedConsumer := infra.NewOrderPlacedConsumer(svc, logger)
	if err := bus.Subscribe("ordering.order.placed", "inventory", placedConsumer.Handle); err != nil {
		logger.Error("order.placed 구독 실패", "err", err)
		os.Exit(1)
	}
	// 주문이 취소되면(재고부족·결제실패 등) 잡아 둔 재고를 되돌린다(보상).
	cancelledConsumer := infra.NewOrderCancelledConsumer(svc, logger)
	if err := bus.Subscribe("ordering.order.cancelled", "inventory", cancelledConsumer.Handle); err != nil {
		logger.Error("order.cancelled 구독 실패", "err", err)
		os.Exit(1)
	}
	// 반품이 요청되면 재고를 다시 채운다.
	returnConsumer := infra.NewReturnRequestedConsumer(svc, logger)
	if err := bus.Subscribe("ordering.order.return_requested", "inventory", returnConsumer.Handle); err != nil {
		logger.Error("order.return_requested 구독 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("inventory 서비스 시작 — order.placed·order.cancelled·order.return_requested 구독 중", "nats", url)

	// 관찰성·probe 용 최소 HTTP 서버(별도 고루틴).
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())
	httpAddr := envOr("HTTP_ADDR", ":8080")
	go func() {
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			logger.Error("헬스/메트릭 서버 종료", "err", err)
		}
	}()

	// 이벤트 소비자는 신호가 올 때까지 그냥 떠 있는다.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("inventory 서비스 종료")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
