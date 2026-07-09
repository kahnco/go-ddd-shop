package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/inventory/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

// inventory 서비스: 주문 컨텍스트의 order.placed 를 구독해 재고를 예약하는 소비자.
// HTTP 를 열지 않는다 — 입구가 오직 이벤트다. 이것이 EDD 서비스의 전형적인 모습이다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	// 데모용 초기 재고.
	repo := infra.NewMemoryStockRepository()
	repo.Seed("prod-A", 10)
	repo.Seed("prod-B", 5)

	publisher := infra.NewNatsEventPublisher(bus, "inventory")
	svc := app.NewReservationService(repo, publisher)
	consumer := infra.NewOrderPlacedConsumer(svc, logger)

	if err := bus.Subscribe("ordering.order.placed", "inventory", consumer.Handle); err != nil {
		logger.Error("구독 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("inventory 서비스 시작 — order.placed 구독 중", "nats", url)

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
