package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/payment/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// NatsEventPublisher 는 결제 컨텍스트의 EventPublisher 포트를 NATS 발행으로 구현.
// 이벤트 이름 자체가 이미 "payment.*" 로 컨텍스트를 담고 있어, 그대로 subject 로 쓴다.
type NatsEventPublisher struct {
	bus *eventbus.Bus
}

func NewNatsEventPublisher(bus *eventbus.Bus) *NatsEventPublisher {
	return &NatsEventPublisher{bus: bus}
}

func (p *NatsEventPublisher) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		subject := e.EventName() // "payment.completed" / "payment.failed"
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
