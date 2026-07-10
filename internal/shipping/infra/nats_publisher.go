package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/shipping/domain"
)

// NatsEventPublisher 는 배송 컨텍스트의 EventPublisher 포트를 NATS 발행으로 구현.
type NatsEventPublisher struct {
	bus *eventbus.Bus
}

func NewNatsEventPublisher(bus *eventbus.Bus) *NatsEventPublisher {
	return &NatsEventPublisher{bus: bus}
}

func (p *NatsEventPublisher) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		subject := e.EventName() // "shipping.dispatched"
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
