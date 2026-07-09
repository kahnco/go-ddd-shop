package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahnco/go-ddd-shop/internal/payment/app"
	"github.com/kahnco/go-ddd-shop/internal/payment/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// payment 서비스: 재고 예약(stock.reserved)을 구독해 결제를 처리하는 소비자.
// 결과(payment.completed / payment.failed)를 다시 발행해 주문 컨텍스트가 이어받게 한다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	repo := infra.NewMemoryPaymentRepository()
	publisher := infra.NewNatsEventPublisher(bus)
	svc := app.NewPaymentService(repo, publisher)
	consumer := infra.NewStockReservedConsumer(svc, logger)

	if err := bus.Subscribe("inventory.stock.reserved", "payment", consumer.Handle); err != nil {
		logger.Error("구독 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("payment 서비스 시작 — stock.reserved 구독 중", "nats", url)

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
	logger.Info("payment 서비스 종료")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
