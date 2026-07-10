package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/shipping/app"
)

// OrderConfirmedConsumer 는 주문 확정(order.confirmed)을 받아 배송 유스케이스로 넘긴다.
type OrderConfirmedConsumer struct {
	svc *app.ShippingService
	log *slog.Logger
}

func NewOrderConfirmedConsumer(svc *app.ShippingService, log *slog.Logger) *OrderConfirmedConsumer {
	return &OrderConfirmedConsumer{svc: svc, log: log}
}

func (c *OrderConfirmedConsumer) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	ctx, span := telemetry.StartSpan(ctx, "consume "+env.Name)
	defer span.End()
	log := c.log.With("correlation_id", telemetry.CorrelationID(ctx))

	var p struct {
		OrderID string `json:"order_id"`
	}
	if err := env.Into(&p); err != nil {
		log.Error("order.confirmed 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("order.confirmed", "decode_error")
		return err
	}

	if err := c.svc.OnOrderConfirmed(ctx, app.DispatchCommand{OrderID: p.OrderID}); err != nil {
		log.Error("배송 처리 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("order.confirmed", "error")
		return err
	}
	log.Info("주문 확정 → 배송 시작", "order", p.OrderID)
	telemetry.RecordEventConsumed("order.confirmed", "ok")
	return nil
}
