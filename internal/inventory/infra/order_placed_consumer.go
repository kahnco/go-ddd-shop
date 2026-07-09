package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// OrderPlacedConsumer 는 주문 컨텍스트의 order.placed 이벤트를 받아
// 재고 예약 유스케이스로 넘기는 인바운드 어댑터.
type OrderPlacedConsumer struct {
	svc *app.ReservationService
	log *slog.Logger
}

func NewOrderPlacedConsumer(svc *app.ReservationService, log *slog.Logger) *OrderPlacedConsumer {
	return &OrderPlacedConsumer{svc: svc, log: log}
}

// orderPlacedPayload 는 order.placed 이벤트의 JSON 을 재고 컨텍스트가 이해하는 모양으로
// "번역"하는 전용 타입이다. 주문 컨텍스트의 도메인 타입을 import 하지 않는 게 핵심 —
// 두 컨텍스트는 오직 이 JSON 계약으로만 연결된다(부패 방지 계층, ACL 의 축소판).
type orderPlacedPayload struct {
	OrderID string `json:"order_id"`
	Total   int64  `json:"total"`
	Items   []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

// Handle 은 봉투 하나를 처리한다. eventbus.Handler 시그니처에 맞는다.
func (c *OrderPlacedConsumer) Handle(env eventbus.Envelope) error {
	// 발행 서비스가 실어 보낸 상관 ID 를 이어받는다. 이 ID 로 로그를 남기면,
	// 주문 서비스의 로그와 같은 ID 로 하나의 주문 흐름을 꿰어 볼 수 있다.
	cid := env.Meta[telemetry.MetaCorrelationID]
	ctx := telemetry.WithCorrelationID(context.Background(), cid)
	log := c.log.With("correlation_id", cid)

	var p orderPlacedPayload
	if err := env.Into(&p); err != nil {
		log.Error("order.placed 디코딩 실패", "err", err)
		telemetry.RecordEventConsumed("order.placed", "decode_error")
		return err
	}

	cmd := app.ReserveForOrderCommand{OrderID: p.OrderID, Amount: p.Total}
	for _, it := range p.Items {
		cmd.Items = append(cmd.Items, app.ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}

	if err := c.svc.OnOrderPlaced(ctx, cmd); err != nil {
		log.Error("재고 예약 처리 실패", "order", p.OrderID, "err", err)
		telemetry.RecordEventConsumed("order.placed", "error")
		return err
	}
	log.Info("order.placed 처리 완료", "order", p.OrderID, "items", len(p.Items))
	telemetry.RecordEventConsumed("order.placed", "ok")
	return nil
}
