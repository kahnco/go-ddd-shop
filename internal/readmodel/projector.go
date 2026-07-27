package readmodel

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// Projector 는 주문 이벤트를 받아 읽기 모델을 갱신한다.
// order.placed 로 뷰를 만들고, 이후 상태 이벤트로 상태만 바꾼다.
//
// 눈여겨볼 점: 이 핸들러는 본질적으로 멱등하다 — 같은 이벤트를 두 번 처리해도
// upsert/set-status 는 같은 결과로 수렴한다. 읽기 모델의 좋은 성질이다.
type Projector struct {
	store Store
	log   *slog.Logger
	hub   *Hub // 있으면 갱신을 SSE 구독자에게 push(없으면 no-op)
}

func NewProjector(store Store, log *slog.Logger) *Projector {
	return &Projector{store: store, log: log}
}

// WithHub 는 SSE 허브를 붙인다 — 프로젝션이 갱신될 때마다 해당 회원에게 push 한다.
func (p *Projector) WithHub(h *Hub) *Projector {
	p.hub = h
	return p
}

// Handle 은 ordering.order.> 로 오는 이벤트를 이름으로 갈라 처리한다.
func (p *Projector) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	_, span := telemetry.StartSpan(ctx, "project "+env.Name)
	defer span.End()

	var orderID, customerID string
	switch env.Name {
	case "order.placed":
		var e struct {
			OrderID    string `json:"order_id"`
			CustomerID string `json:"customer_id"`
			Total      int64  `json:"total"`
			Channel    string `json:"channel"` // v2. 옛 이벤트엔 없지만 업캐스터가 채워 준다.
			Items      []Item `json:"items"`
		}
		if err := env.Into(&e); err != nil {
			return err
		}
		p.store.Upsert(OrderView{
			OrderID: e.OrderID, CustomerID: e.CustomerID,
			Status: "PLACED", Total: e.Total, Channel: e.Channel, Items: e.Items,
		})
		orderID, customerID = e.OrderID, e.CustomerID
	case "order.paid":
		orderID = p.setStatus(env, "PAID")
	case "order.confirmed":
		orderID = p.setStatus(env, "CONFIRMED")
	case "order.shipped":
		orderID = p.setStatus(env, "SHIPPED")
	case "order.cancelled":
		orderID = p.setStatus(env, "CANCELLED")
	case "order.return_requested":
		orderID = p.setStatus(env, "RETURN_REQUESTED")
	case "order.refunded":
		orderID = p.setStatus(env, "REFUNDED")
	default:
		return nil // 관심 없는 이벤트는 무시
	}
	telemetry.RecordEventConsumed(env.Name, "ok")
	p.notify(orderID, customerID) // SSE 구독자에게 최신 주문 목록 push
	return nil
}

func (p *Projector) setStatus(env eventbus.Envelope, status string) string {
	var e struct {
		OrderID string `json:"order_id"`
	}
	if err := env.Into(&e); err != nil {
		return ""
	}
	p.store.SetStatus(e.OrderID, status)
	return e.OrderID
}

// notify 는 갱신된 주문의 주인에게 최신 주문 목록을 SSE 로 밀어 준다.
func (p *Projector) notify(orderID, customerID string) {
	if p.hub == nil || orderID == "" {
		return
	}
	if customerID == "" {
		if v, ok := p.store.Get(orderID); ok {
			customerID = v.CustomerID
		}
	}
	if customerID != "" {
		p.hub.Publish(customerID, ordersJSON(p.store, customerID))
	}
}
