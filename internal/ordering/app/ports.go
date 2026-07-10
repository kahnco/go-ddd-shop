package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// 애플리케이션 계층이 바깥 세계에 요구하는 "포트"들.
// 도메인의 OrderRepository 포트와 함께, 구현은 인프라 계층이 채운다.

// EventPublisher 는 도메인 이벤트를 밖으로 발행하는 포트.
// 3편에서는 로그로 찍는 단순 구현을 쓰고, 4편에서 메시지 브로커(NATS)로 바꾼다.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}

// IDGenerator 는 새 주문 식별자를 만드는 포트.
// 포트로 둔 덕에, 테스트에서는 고정 ID 를 주입해 결과를 결정적으로 검증할 수 있다.
type IDGenerator interface {
	NewOrderID() domain.OrderID
}

// ProductPriceLookup 은 상품의 "권위 있는 가격"을 돌려주는 포트.
// 가격은 클라이언트가 아니라 카탈로그가 정한다 — 이 포트로 그 가격을 가져온다.
// 구현은 카탈로그 이벤트로 갱신되는 로컬 프로젝션(읽기 모델)이 채운다.
// 카탈로그에 없는 상품이면 domain.ErrUnknownProduct 를 돌려준다.
type ProductPriceLookup interface {
	PriceOf(ctx context.Context, productID domain.ProductID) (domain.Money, error)
}
