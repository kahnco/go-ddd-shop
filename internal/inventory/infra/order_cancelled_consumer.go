package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// OrderCancelledConsumer 는 주문 취소(order.cancelled)를 받아 예약 재고를 되돌린다(보상).
type OrderCancelledConsumer struct {
	svc *app.ReservationService
	log *slog.Logger
}

func NewOrderCancelledConsumer(svc *app.ReservationService, log *slog.Logger) *OrderCancelledConsumer {
	return &OrderCancelledConsumer{svc: svc, log: log}
}

func (c *OrderCancelledConsumer) Handle(env eventbus.Envelope) error {
	cid := env.Meta[telemetry.MetaCorrelationID]
	ctx := telemetry.WithCorrelationID(context.Background(), cid)
	log := c.log.With("correlation_id", cid)

	var p struct {
		OrderID string `json:"order_id"`
	}
	if err := env.Into(&p); err != nil {
		log.Error("order.cancelled 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("order.cancelled", "decode_error")
		return err
	}

	if err := c.svc.OnOrderCancelled(ctx, p.OrderID); err != nil {
		log.Error("예약 복원 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("order.cancelled", "error")
		return err
	}
	log.Info("주문 취소 → 예약 재고 복원", "order", p.OrderID)
	telemetry.RecordEventConsumed("order.cancelled", "ok")
	return nil
}
