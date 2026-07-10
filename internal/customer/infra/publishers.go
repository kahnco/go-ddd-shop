package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// LogPublisher 는 이벤트를 로그로 찍는 어댑터(NATS 없이 단독 실행용).
type LogPublisher struct{ logger *slog.Logger }

func NewLogPublisher(logger *slog.Logger) *LogPublisher { return &LogPublisher{logger: logger} }

func (p *LogPublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		p.logger.Info("domain event published", "event", e.EventName())
	}
	return nil
}

// NatsEventPublisher 는 회원 이벤트를 NATS 로 발행하는 어댑터.
// 이벤트 이름이 이미 "customer.*" 라 그대로 subject 로 쓴다(CUSTOMER 스트림이 받는다).
type NatsEventPublisher struct {
	bus *eventbus.Bus
}

func NewNatsEventPublisher(bus *eventbus.Bus) *NatsEventPublisher {
	return &NatsEventPublisher{bus: bus}
}

func (p *NatsEventPublisher) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		env, err := eventbus.NewEnvelope(e.EventName(), e)
		if err != nil {
			return err
		}
		env.Meta = telemetry.MetaFromContext(ctx)
		if err := p.bus.Publish(e.EventName(), env); err != nil {
			return err
		}
		telemetry.RecordEventPublished(e.EventName())
	}
	return nil
}
