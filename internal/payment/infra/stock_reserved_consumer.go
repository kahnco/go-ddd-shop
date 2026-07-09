package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/payment/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// StockReservedConsumer 는 재고 컨텍스트의 stock.reserved 를 받아 결제 유스케이스로 넘긴다.
type StockReservedConsumer struct {
	svc *app.PaymentService
	log *slog.Logger
}

func NewStockReservedConsumer(svc *app.PaymentService, log *slog.Logger) *StockReservedConsumer {
	return &StockReservedConsumer{svc: svc, log: log}
}

// stockReservedPayload 는 stock.reserved 이벤트를 결제 컨텍스트가 이해하는 모양으로 번역한다.
type stockReservedPayload struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

func (c *StockReservedConsumer) Handle(env eventbus.Envelope) error {
	cid := env.Meta[telemetry.MetaCorrelationID]
	ctx := telemetry.WithCorrelationID(context.Background(), cid)
	log := c.log.With("correlation_id", cid)

	var p stockReservedPayload
	if err := env.Into(&p); err != nil {
		log.Error("stock.reserved 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("stock.reserved", "decode_error")
		return err
	}

	cmd := app.ProcessPaymentCommand{OrderID: p.OrderID, Amount: p.Amount}
	if err := c.svc.OnStockReserved(ctx, cmd); err != nil {
		log.Error("결제 처리 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("stock.reserved", "error")
		return err
	}
	log.Info("결제 처리 완료", "order", p.OrderID, "amount", p.Amount)
	telemetry.RecordEventConsumed("stock.reserved", "ok")
	return nil
}
