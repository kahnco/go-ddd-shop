package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// EventPublisher 는 재고 컨텍스트가 자신의 이벤트를 밖으로 발행하는 포트.
// 주문 컨텍스트의 같은 이름 포트와는 별개다(컨텍스트는 코드를 공유하지 않는다).
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}
