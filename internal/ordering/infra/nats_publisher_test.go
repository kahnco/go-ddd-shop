package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// OrderPlaced 를 발행하면, 올바른 subject 로 항목·총액이 실린 JSON 이 나가는지 본다.
func TestNatsEventPublisher_OrderPlaced_를_구독자가_받는다(t *testing.T) {
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

	got := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.placed", "test", func(env eventbus.Envelope) error {
		got <- env
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	pub := infra.NewNatsEventPublisher(bus, "ordering")
	event := domain.OrderPlaced{
		OrderID:    "order-1",
		CustomerID: "cust-1",
		Total:      mustMoney(t, 5000),
		Items: []domain.OrderPlacedItem{
			{ProductID: "prod-A", Quantity: 2},
			{ProductID: "prod-B", Quantity: 1},
		},
	}
	if err := pub.Publish(context.Background(), event); err != nil {
		t.Fatal(err)
	}

	select {
	case env := <-got:
		if env.Name != "order.placed" {
			t.Fatalf("이벤트 이름 = order.placed 여야 하는데 %s", env.Name)
		}
		// 받는 쪽은 자기만의 타입으로 푼다(발행 도메인 타입에 의존하지 않음).
		var payload struct {
			OrderID string `json:"order_id"`
			Total   int64  `json:"total"`
			Items   []struct {
				ProductID string `json:"product_id"`
				Quantity  int    `json:"quantity"`
			} `json:"items"`
		}
		if err := env.Into(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.OrderID != "order-1" || payload.Total != 5000 {
			t.Fatalf("payload 불일치: %+v", payload)
		}
		if len(payload.Items) != 2 || payload.Items[0].ProductID != "prod-A" || payload.Items[0].Quantity != 2 {
			t.Fatalf("항목 불일치: %+v", payload.Items)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("타임아웃: 이벤트를 받지 못함")
	}
}

func mustMoney(t *testing.T, amount int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(amount)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", amount, err)
	}
	return m
}
