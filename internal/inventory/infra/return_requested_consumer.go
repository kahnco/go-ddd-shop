package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// ReturnRequestedConsumer 는 주문의 order.return_requested 를 받아 반품 재입고 유스케이스로 넘긴다.
type ReturnRequestedConsumer struct {
	svc *app.ReservationService
	log *slog.Logger
}

func NewReturnRequestedConsumer(svc *app.ReservationService, log *slog.Logger) *ReturnRequestedConsumer {
	return &ReturnRequestedConsumer{svc: svc, log: log}
}

func (c *ReturnRequestedConsumer) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	ctx, span := telemetry.StartSpan(ctx, "consume "+env.Name)
	defer span.End()
	log := c.log.With("correlation_id", telemetry.CorrelationID(ctx))

	var p struct {
		OrderID string `json:"order_id"`
		Items   []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := env.Into(&p); err != nil {
		log.Error("order.return_requested 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("order.return_requested", "decode_error")
		return err
	}

	cmd := app.ReserveForOrderCommand{OrderID: p.OrderID}
	for _, it := range p.Items {
		cmd.Items = append(cmd.Items, app.ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	if err := c.svc.OnReturnRequested(ctx, cmd); err != nil {
		log.Error("반품 재입고 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("order.return_requested", "error")
		return err
	}
	log.Info("반품 요청 → 재고 재입고", "order", p.OrderID, "items", len(p.Items))
	telemetry.RecordEventConsumed("order.return_requested", "ok")
	return nil
}
