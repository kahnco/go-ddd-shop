package infra

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// NoopPublisher 는 아무것도 하지 않는 EventPublisher.
// 아웃박스 모드에서 쓴다 — 이벤트는 저장소 트랜잭션으로 아웃박스에 적재되고
// 릴레이가 발행하므로, 유스케이스의 "직접 발행"은 비활성화한다.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, ...domain.DomainEvent) error { return nil }
