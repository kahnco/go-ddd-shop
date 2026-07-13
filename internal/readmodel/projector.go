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
}

func NewProjector(store Store, log *slog.Logger) *Projector {
	return &Projector{store: store, log: log}
}

// Handle 은 ordering.order.> 로 오는 이벤트를 이름으로 갈라 처리한다.
func (p *Projector) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	_, span := telemetry.StartSpan(ctx, "project "+env.Name)
	defer span.End()

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
	case "order.paid":
		p.setStatus(env, "PAID")
	case "order.confirmed":
		p.setStatus(env, "CONFIRMED")
	case "order.shipped":
		p.setStatus(env, "SHIPPED")
	case "order.cancelled":
		p.setStatus(env, "CANCELLED")
	case "order.return_requested":
		p.setStatus(env, "RETURN_REQUESTED")
	case "order.refunded":
		p.setStatus(env, "REFUNDED")
	default:
		return nil // 관심 없는 이벤트는 무시
	}
	telemetry.RecordEventConsumed(env.Name, "ok")
	return nil
}

func (p *Projector) setStatus(env eventbus.Envelope, status string) {
	var e struct {
		OrderID string `json:"order_id"`
	}
	if err := env.Into(&e); err != nil {
		return
	}
	p.store.SetStatus(e.OrderID, status)
}
