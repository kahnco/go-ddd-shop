package infra

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"

	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// OrderSagaConsumer 는 주문 컨텍스트가 다른 컨텍스트의 이벤트에 반응하는 인바운드 어댑터.
// 여기서 주문의 여정(placed → paid → confirmed, 또는 cancelled)이 이벤트로 이어진다.
type OrderSagaConsumer struct {
	svc *app.OrderService
	log *slog.Logger
}

func NewOrderSagaConsumer(svc *app.OrderService, log *slog.Logger) *OrderSagaConsumer {
	return &OrderSagaConsumer{svc: svc, log: log}
}

type orderRef struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}

// prepare 는 봉투에서 상관 ID·trace 컨텍스트·주문 ID 를 뽑아 ctx·span·logger 를 갖춘다(핸들러 공통).
// 반환한 span 은 호출자가 defer span.End() 로 닫아야 한다.
func (c *OrderSagaConsumer) prepare(env eventbus.Envelope) (context.Context, trace.Span, *slog.Logger, orderRef, error) {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	ctx, span := telemetry.StartSpan(ctx, "consume "+env.Name)
	log := c.log.With("correlation_id", telemetry.CorrelationID(ctx))
	var ref orderRef
	err := env.Into(&ref)
	return ctx, span, log, ref, err
}

// OnPaymentCompleted 는 결제 완료(payment.completed)에 반응해 주문을 확정한다.
func (c *OrderSagaConsumer) OnPaymentCompleted(env eventbus.Envelope) error {
	ctx, span, log, ref, err := c.prepare(env)
	defer span.End()
	if err != nil {
		telemetry.RecordEventConsumed("payment.completed", "decode_error")
		return err
	}
	if err := c.svc.ConfirmPaidOrder(ctx, domain.OrderID(ref.OrderID)); err != nil {
		log.Error("주문 확정 실패", "order", ref.OrderID, "err", err)
		telemetry.RecordEventConsumed("payment.completed", "error")
		return err
	}
	log.Info("결제 완료 → 주문 확정", "order", ref.OrderID)
	telemetry.RecordEventConsumed("payment.completed", "ok")
	return nil
}

// OnStockReservationFailed 는 재고 부족(stock.reservation_failed)에 반응해 주문을 취소한다.
func (c *OrderSagaConsumer) OnStockReservationFailed(env eventbus.Envelope) error {
	ctx, span, log, ref, err := c.prepare(env)
	defer span.End()
	if err != nil {
		telemetry.RecordEventConsumed("stock.reservation_failed", "decode_error")
		return err
	}
	if err := c.svc.CancelOrder(ctx, domain.OrderID(ref.OrderID)); err != nil {
		log.Error("주문 취소 실패", "order", ref.OrderID, "err", err)
		telemetry.RecordEventConsumed("stock.reservation_failed", "error")
		return err
	}
	log.Info("재고 부족 → 주문 취소", "order", ref.OrderID, "reason", ref.Reason)
	telemetry.RecordEventConsumed("stock.reservation_failed", "ok")
	return nil
}

// OnPaymentFailed 는 결제 실패(payment.failed)에 반응해 주문을 취소한다.
// 취소 이벤트를 받은 재고 컨텍스트가 예약한 재고까지 되돌린다(완전한 보상).
func (c *OrderSagaConsumer) OnPaymentFailed(env eventbus.Envelope) error {
	ctx, span, log, ref, err := c.prepare(env)
	defer span.End()
	if err != nil {
		telemetry.RecordEventConsumed("payment.failed", "decode_error")
		return err
	}
	if err := c.svc.CancelOrder(ctx, domain.OrderID(ref.OrderID)); err != nil {
		log.Error("주문 취소 실패", "order", ref.OrderID, "err", err)
		telemetry.RecordEventConsumed("payment.failed", "error")
		return err
	}
	log.Info("결제 실패 → 주문 취소", "order", ref.OrderID, "reason", ref.Reason)
	telemetry.RecordEventConsumed("payment.failed", "ok")
	return nil
}

// OnPaymentRefunded 는 환불 완료(payment.refunded)에 반응해 주문을 환불완료로 전이한다.
func (c *OrderSagaConsumer) OnPaymentRefunded(env eventbus.Envelope) error {
	ctx, span, log, ref, err := c.prepare(env)
	defer span.End()
	if err != nil {
		telemetry.RecordEventConsumed("payment.refunded", "decode_error")
		return err
	}
	if err := c.svc.MarkOrderRefunded(ctx, domain.OrderID(ref.OrderID)); err != nil {
		log.Error("주문 환불완료 전이 실패", "order", ref.OrderID, "err", err)
		telemetry.RecordEventConsumed("payment.refunded", "error")
		return err
	}
	log.Info("환불 완료 → 주문 환불완료", "order", ref.OrderID)
	telemetry.RecordEventConsumed("payment.refunded", "ok")
	return nil
}

// OnShipmentDispatched 는 배송 시작(shipping.dispatched)에 반응해 주문을 배송중으로 전이한다.
func (c *OrderSagaConsumer) OnShipmentDispatched(env eventbus.Envelope) error {
	ctx, span, log, ref, err := c.prepare(env)
	defer span.End()
	if err != nil {
		telemetry.RecordEventConsumed("shipping.dispatched", "decode_error")
		return err
	}
	if err := c.svc.ShipOrder(ctx, domain.OrderID(ref.OrderID)); err != nil {
		log.Error("주문 배송 전이 실패", "order", ref.OrderID, "err", err)
		telemetry.RecordEventConsumed("shipping.dispatched", "error")
		return err
	}
	log.Info("배송 시작 → 주문 배송중", "order", ref.OrderID)
	telemetry.RecordEventConsumed("shipping.dispatched", "ok")
	return nil
}
