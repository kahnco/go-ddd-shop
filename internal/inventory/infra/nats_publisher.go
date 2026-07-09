package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
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

func (p *NatsEventPublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		subject := p.prefix + "." + e.EventName()
		env, err := eventbus.NewEnvelope(e.EventName(), e)
		if err != nil {
			return err
		}
		if err := p.bus.Publish(subject, env); err != nil {
			return err
		}
	}
	return nil
}
