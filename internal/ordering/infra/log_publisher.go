package infra

import (
	"context"
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// LogPublisher 는 app.EventPublisher 포트를 "로그로 찍기"로 구현한 임시 어댑터.
// 4편에서 이걸 NATS 발행 어댑터로 갈아끼우면, 다른 서비스가 이벤트를 받게 된다.
// (역시 애플리케이션 코드는 한 줄도 안 바꾼다.)
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
