package domain

import "context"

// OrderRepository 는 주문 저장소의 "포트(port)".
// 인터페이스만 도메인에 두고, 실제 구현(DB)은 인프라 계층에 둔다(의존성 역전).
// 그래서 도메인은 DB가 무엇인지 전혀 모른 채로 순수하게 테스트할 수 있다.
// 실제 구현은 3편(인프라 계층)에서.
type OrderRepository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id OrderID) (*Order, error)
}
