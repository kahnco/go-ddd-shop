package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/shipping/app"
	"github.com/kahnco/go-ddd-shop/internal/shipping/infra"
)

// shipping 서비스: 주문 확정(order.confirmed)을 구독해 배송을 시작하는 소비자.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shutdown, _ := telemetry.InitTracer(context.Background(), "shipping")
	defer func() { _ = shutdown(context.Background()) }()

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	repo := infra.NewMemoryShipmentRepository()
	publisher := infra.NewNatsEventPublisher(bus)
	svc := app.NewShippingService(repo, publisher)
	consumer := infra.NewOrderConfirmedConsumer(svc, logger)

	if err := bus.Subscribe("ordering.order.confirmed", "shipping", consumer.Handle); err != nil {
		logger.Error("구독 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("shipping 서비스 시작 — order.confirmed 구독 중", "nats", url)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())
	go func() {
		if err := http.ListenAndServe(envOr("HTTP_ADDR", ":8080"), mux); err != nil {
			logger.Error("헬스/메트릭 서버 종료", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shipping 서비스 종료")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
