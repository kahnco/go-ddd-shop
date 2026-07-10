package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
)

// LogPublisher 는 EventPublisher 포트를 로그로 구현한 어댑터(NATS 없이 단독 실행용).
type LogPublisher struct {
	logger *slog.Logger
}

func NewLogPublisher(logger *slog.Logger) *LogPublisher {
	return &LogPublisher{logger: logger}
}

func (p *LogPublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	for _, e := range events {
		p.logger.Info("domain event published", "event", e.EventName())
	}
	return nil
}
