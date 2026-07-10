// Package integration 은 컨텍스트를 가로지르는 이벤트 흐름을 실제(임베디드) NATS 위에서 검증한다.
// 주문 컨텍스트가 발행한 OrderPlaced 가 재고 컨텍스트로 전달돼 예약이 일어나는,
// EDD 의 핵심 경로를 종단 간(end-to-end)으로 확인한다.
package integration

import (
	"context"
	"testing"
	"time"

	orderingdomain "github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	orderinginfra "github.com/kahnco/go-ddd-shop/internal/ordering/infra"

	invapp "github.com/kahnco/go-ddd-shop/internal/inventory/app"
	invinfra "github.com/kahnco/go-ddd-shop/internal/inventory/infra"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// 재고 컨텍스트를 구성하고 order.placed 를 구독시킨다. 결과를 관찰하려고
// 재고가 내보내는 stock.* 이벤트를 test 가 함께 구독한다.
func wireInventory(t *testing.T, bus *eventbus.Bus, seed map[string]int) *invinfra.MemoryStockRepository {
	t.Helper()
	repo := invinfra.NewMemoryStockRepository()
	for id, n := range seed {
		repo.Seed(invDomainProductID(id), n)
	}
	reservations := invinfra.NewMemoryReservationRepository()
	pub := invinfra.NewNatsEventPublisher(bus, "inventory")
	svc := invapp.NewReservationService(repo, reservations, pub)

	placed := invinfra.NewOrderPlacedConsumer(svc, discardLogger())
	if err := bus.Subscribe("ordering.order.placed", "inventory", placed.Handle); err != nil {
		t.Fatalf("order.placed 구독: %v", err)
	}
	cancelled := invinfra.NewOrderCancelledConsumer(svc, discardLogger())
	if err := bus.Subscribe("ordering.order.cancelled", "inventory", cancelled.Handle); err != nil {
		t.Fatalf("order.cancelled 구독: %v", err)
	}
	return repo
}

func TestOrderPlaced_흐르면_재고가_예약된다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	bus, err := eventbus.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	// 재고 컨텍스트: prod-A 10개, prod-B 5개.
	repo := wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 5})

	// 재고가 내보내는 결과 이벤트를 관찰(동기화용).
	reserved := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("inventory.stock.reserved", "test", func(e eventbus.Envelope) error {
		reserved <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// 주문 컨텍스트: 유스케이스가 하듯 OrderPlaced 를 발행한다.
	orderingPub := orderinginfra.NewNatsEventPublisher(bus, "ordering")
	event := orderingdomain.OrderPlaced{
		OrderID:    "order-1",
		CustomerID: "cust-1",
		Total:      mustMoney(t, 5000),
		Items: []orderingdomain.OrderPlacedItem{
			{ProductID: "prod-A", Quantity: 2},
			{ProductID: "prod-B", Quantity: 1},
		},
	}
	if err := orderingPub.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	// 재고 컨텍스트가 처리해 StockReserved 를 낼 때까지 기다린다.
	select {
	case <-reserved:
	case <-time.After(3 * time.Second):
		t.Fatal("타임아웃: 재고 예약 이벤트를 받지 못함")
	}

	a, _ := repo.FindByProduct(context.Background(), invDomainProductID("prod-A"))
	b, _ := repo.FindByProduct(context.Background(), invDomainProductID("prod-B"))
	if a.Available() != 8 || b.Available() != 4 {
		t.Fatalf("예약 후 재고 A=8,B=4 여야 하는데 A=%d,B=%d", a.Available(), b.Available())
	}
}

func TestOrderPlaced_재고부족이면_실패이벤트가_흐른다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	bus, err := eventbus.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	// prod-B 재고가 0이라 예약이 실패해야 한다.
	repo := wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 0})

	failed := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("inventory.stock.reservation_failed", "test", func(e eventbus.Envelope) error {
		failed <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	orderingPub := orderinginfra.NewNatsEventPublisher(bus, "ordering")
	event := orderingdomain.OrderPlaced{
		OrderID: "order-2",
		Items: []orderingdomain.OrderPlacedItem{
			{ProductID: "prod-A", Quantity: 2},
			{ProductID: "prod-B", Quantity: 1},
		},
	}
	if err := orderingPub.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	select {
	case <-failed:
	case <-time.After(3 * time.Second):
		t.Fatal("타임아웃: 예약 실패 이벤트를 받지 못함")
	}

	// 보상으로 prod-A 는 원상복구(10)돼 있어야 한다.
	a, _ := repo.FindByProduct(context.Background(), invDomainProductID("prod-A"))
	if a.Available() != 10 {
		t.Fatalf("보상 후 prod-A 재고 = 10 이어야 하는데 %d", a.Available())
	}
}

func mustMoney(t *testing.T, amount int64) orderingdomain.Money {
	t.Helper()
	m, err := orderingdomain.NewMoney(amount)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}
