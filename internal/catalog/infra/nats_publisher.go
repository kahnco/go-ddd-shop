package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// NatsEventPublisher 는 카탈로그의 EventPublisher 포트를 NATS 발행으로 구현.
type NatsEventPublisher struct {
	bus    *eventbus.Bus
	prefix string // "catalog" → "catalog.product.added"
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
