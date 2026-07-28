package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// NatsEventPublisher 는 프로모션 컨텍스트의 app.EventPublisher 포트를 NATS 발행으로 구현.
// prefix="promotion" → 주제 "promotion.winner_determined".
type NatsEventPublisher struct {
	bus    *eventbus.Bus
	prefix string
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
		env.Meta = telemetry.MetaFromContext(ctx)
		if err := p.bus.Publish(subject, env); err != nil {
			return err
		}
		telemetry.RecordEventPublished(e.EventName())
	}
	return nil
}

var _ interface {
	Publish(context.Context, ...domain.DomainEvent) error
} = (*NatsEventPublisher)(nil)
