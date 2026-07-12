package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/payment/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// ReturnRequestedConsumer 는 주문의 order.return_requested 를 받아 환불 유스케이스로 넘긴다.
type ReturnRequestedConsumer struct {
	svc *app.PaymentService
	log *slog.Logger
}

func NewReturnRequestedConsumer(svc *app.PaymentService, log *slog.Logger) *ReturnRequestedConsumer {
	return &ReturnRequestedConsumer{svc: svc, log: log}
}

func (c *ReturnRequestedConsumer) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	ctx, span := telemetry.StartSpan(ctx, "consume "+env.Name)
	defer span.End()
	log := c.log.With("correlation_id", telemetry.CorrelationID(ctx))

	var p struct {
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
	}
	if err := env.Into(&p); err != nil {
		log.Error("order.return_requested 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("order.return_requested", "decode_error")
		return err
	}

	if err := c.svc.OnReturnRequested(ctx, app.RefundCommand{OrderID: p.OrderID, Amount: p.Amount}); err != nil {
		log.Error("환불 처리 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("order.return_requested", "error")
		return err
	}
	log.Info("반품 요청 → 환불 처리", "order", p.OrderID, "amount", p.Amount)
	telemetry.RecordEventConsumed("order.return_requested", "ok")
	return nil
}
