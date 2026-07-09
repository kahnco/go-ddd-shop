package domain

// 도메인 이벤트: 도메인에서 "일어난 사실"을 과거형으로 표현한다.
// 애그리거트가 상태를 바꿀 때 이벤트를 기록(record)하고, 나중에 밖으로 발행된다.
// 다른 bounded context(재고·결제·배송)는 이 이벤트를 구독해 반응한다(EDD).
//
// json 태그는 브로커로 오가는 "이벤트 계약(schema)"이다. 다른 컨텍스트는 이 JSON 모양에만
// 의존하지, 이 Go 타입을 공유하지 않는다. 그래서 필드 이름을 안정적인 snake_case 로 고정한다.

// DomainEvent 는 모든 도메인 이벤트가 구현하는 최소 인터페이스.
type DomainEvent interface {
	EventName() string
}

// OrderPlacedItem — OrderPlaced 이벤트가 실어 나르는 항목 정보.
// 재고 컨텍스트가 "무엇을 몇 개 예약할지" 알려면 이 데이터가 이벤트에 실려야 한다
// (event-carried state transfer). 이벤트를 받는 쪽이 원본 주문을 되묻지 않아도 되게.
type OrderPlacedItem struct {
	ProductID ProductID `json:"product_id"`
	Quantity  int       `json:"quantity"`
}

// OrderPlaced — 주문이 생성됨. 재고 컨텍스트가 구독해 재고를 예약한다.
type OrderPlaced struct {
	OrderID    OrderID           `json:"order_id"`
	CustomerID CustomerID        `json:"customer_id"`
	Total      Money             `json:"total"`
	Items      []OrderPlacedItem `json:"items"`
}

func (OrderPlaced) EventName() string { return "order.placed" }

// OrderPaid — 결제 완료로 주문이 결제됨.
type OrderPaid struct {
	OrderID OrderID `json:"order_id"`
}

func (OrderPaid) EventName() string { return "order.paid" }

// OrderConfirmed — 주문 확정. 배송 컨텍스트가 구독한다.
type OrderConfirmed struct {
	OrderID OrderID `json:"order_id"`
}

func (OrderConfirmed) EventName() string { return "order.confirmed" }

// OrderShipped — 배송 시작.
type OrderShipped struct {
	OrderID OrderID `json:"order_id"`
}

func (OrderShipped) EventName() string { return "order.shipped" }

// OrderCancelled — 주문 취소. 재고 컨텍스트가 구독해 예약 재고를 복원한다(보상).
type OrderCancelled struct {
	OrderID OrderID `json:"order_id"`
}

func (OrderCancelled) EventName() string { return "order.cancelled" }
