package domain

// 도메인 이벤트: 도메인에서 "일어난 사실"을 과거형으로 표현한다.
// 애그리거트가 상태를 바꿀 때 이벤트를 기록(record)하고, 나중에 밖으로 발행된다.
// 다른 bounded context(재고·결제·배송)는 이 이벤트를 구독해 반응한다(EDD).

// DomainEvent 는 모든 도메인 이벤트가 구현하는 최소 인터페이스.
type DomainEvent interface {
	EventName() string
}

// OrderPlaced — 주문이 생성됨. 재고 컨텍스트가 구독해 재고를 예약한다.
type OrderPlaced struct {
	OrderID    OrderID
	CustomerID CustomerID
	Total      Money
}

func (OrderPlaced) EventName() string { return "order.placed" }

// OrderPaid — 결제 완료로 주문이 결제됨.
type OrderPaid struct{ OrderID OrderID }

func (OrderPaid) EventName() string { return "order.paid" }

// OrderConfirmed — 주문 확정. 배송 컨텍스트가 구독한다.
type OrderConfirmed struct{ OrderID OrderID }

func (OrderConfirmed) EventName() string { return "order.confirmed" }

// OrderShipped — 배송 시작.
type OrderShipped struct{ OrderID OrderID }

func (OrderShipped) EventName() string { return "order.shipped" }

// OrderCancelled — 주문 취소. 재고 컨텍스트가 구독해 예약 재고를 복원한다(보상).
type OrderCancelled struct{ OrderID OrderID }

func (OrderCancelled) EventName() string { return "order.cancelled" }
