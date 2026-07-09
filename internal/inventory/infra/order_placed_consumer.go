package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
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
	Items   []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	} `json:"items"`
}

// Handle 은 봉투 하나를 처리한다. eventbus.Handler 시그니처에 맞는다.
func (c *OrderPlacedConsumer) Handle(env eventbus.Envelope) error {
	var p orderPlacedPayload
	if err := env.Into(&p); err != nil {
		c.log.Error("order.placed 디코딩 실패", "err", err)
		return err
	}

	cmd := app.ReserveForOrderCommand{OrderID: p.OrderID}
	for _, it := range p.Items {
		cmd.Items = append(cmd.Items, app.ReservationItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}

	if err := c.svc.OnOrderPlaced(context.Background(), cmd); err != nil {
		c.log.Error("재고 예약 처리 실패", "order", p.OrderID, "err", err)
		return err
	}
	c.log.Info("order.placed 처리 완료", "order", p.OrderID, "items", len(p.Items))
	return nil
}
