package infra_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
	"github.com/kahnco/go-ddd-shop/internal/inventory/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, ...domain.DomainEvent) error { return nil }

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// 같은 이벤트 ID 가 두 번 전달돼도 재고는 한 번만 차감되는지(멱등) 본다.
func TestOrderPlacedConsumer_중복이벤트는_한번만_예약한다(t *testing.T) {
	stock := infra.NewMemoryStockRepository()
	stock.Seed("prod-A", 10)
	reservations := infra.NewMemoryReservationRepository()
	svc := app.NewReservationService(stock, reservations, nopPublisher{})
	consumer := infra.NewOrderPlacedConsumer(svc, discardLogger())

	type item struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	env, err := eventbus.NewEnvelope("order.placed", struct {
		OrderID string `json:"order_id"`
		Total   int64  `json:"total"`
		Items   []item `json:"items"`
	}{OrderID: "order-1", Total: 2000, Items: []item{{ProductID: "prod-A", Quantity: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	env.ID = "evt-1" // 같은 이벤트 ID

	if err := consumer.Handle(env); err != nil {
		t.Fatalf("첫 처리: %v", err)
	}
	if err := consumer.Handle(env); err != nil { // 중복 전달
		t.Fatalf("중복 처리: %v", err)
	}

	a, _ := stock.FindByProduct(context.Background(), "prod-A")
	if a.Available() != 8 {
		t.Fatalf("중복이라도 재고는 한 번만(8) 차감돼야 하는데 %d", a.Available())
	}
}
