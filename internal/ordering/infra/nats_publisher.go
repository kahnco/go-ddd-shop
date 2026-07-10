package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// NatsEventPublisher 는 app.EventPublisher 포트를 NATS 발행으로 구현한 어댑터.
// 3편의 LogPublisher 를 이걸로 갈아끼우면, 도메인 이벤트가 실제로 브로커에 실려
// 다른 컨텍스트로 전달된다. 애플리케이션·도메인 코드는 한 줄도 바뀌지 않는다.
type NatsEventPublisher struct {
	bus    *eventbus.Bus
	prefix string // subject 접두사. 예: "ordering" → "ordering.order.placed"
}

func NewNatsEventPublisher(bus *eventbus.Bus, prefix string) *NatsEventPublisher {
	return &NatsEventPublisher{bus: bus, prefix: prefix}
}

func (p *NatsEventPublisher) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		subject := p.prefix + "." + e.EventName() // ordering.order.placed
		env, err := eventbus.NewEnvelope(e.EventName(), e)
		if err != nil {
			return err
		}
		// 상관 ID + trace 컨텍스트(W3C traceparent)를 이벤트에 실어, 소비 서비스까지 흐름을 잇는다.
		env.Meta = telemetry.MetaFromContext(ctx)
		if err := p.bus.Publish(subject, env); err != nil {
			return err
		}
		telemetry.RecordEventPublished(e.EventName())
	}
	return nil
}
