package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// NatsEventPublisher 는 재고 컨텍스트의 app.EventPublisher 포트를 NATS 발행으로 구현.
// 주문 컨텍스트의 같은 이름 어댑터와 코드가 닮았지만, 컨텍스트가 다르므로 별개로 둔다.
type NatsEventPublisher struct {
	bus    *eventbus.Bus
	prefix string // "inventory" → "inventory.stock.reserved"
}

func NewNatsEventPublisher(bus *eventbus.Bus, prefix string) *NatsEventPublisher {
	return &NatsEventPublisher{bus: bus, prefix: prefix}
}

func (p *NatsEventPublisher) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		subject := p.prefix + "." + e.EventName()
		env, err := eventbus.NewEnvelope(e.EventName(), e)
		if err != nil {
			return err
		}
		// 소비하며 이어받은 상관 ID 를, 이 컨텍스트가 내보내는 이벤트에도 계속 실어 보낸다.
		if cid := telemetry.CorrelationID(ctx); cid != "" {
			env.Meta = map[string]string{telemetry.MetaCorrelationID: cid}
		}
		if err := p.bus.Publish(subject, env); err != nil {
			return err
		}
		telemetry.RecordEventPublished(e.EventName())
	}
	return nil
}
