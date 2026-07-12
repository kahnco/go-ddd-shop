package domain

// 결제 컨텍스트가 발행하는 도메인 이벤트. 주문 컨텍스트가 이 JSON 계약을 구독한다.

type DomainEvent interface {
	EventName() string
}

// PaymentCompleted — 결제 성공. 주문 컨텍스트가 구독해 주문을 확정한다.
type PaymentCompleted struct {
	OrderID OrderID `json:"order_id"`
	Amount  int64   `json:"amount"`
}

func (PaymentCompleted) EventName() string { return "payment.completed" }

// PaymentFailed — 결제 실패. 주문 컨텍스트가 구독해 주문을 취소하고,
// 재고 컨텍스트는 예약을 되돌린다(보상). 2편에서 이 경로를 완성한다.
type PaymentFailed struct {
	OrderID OrderID `json:"order_id"`
	Reason  string  `json:"reason"`
}

func (PaymentFailed) EventName() string { return "payment.failed" }

// PaymentRefunded — 환불 완료. 주문 컨텍스트가 구독해 주문을 환불완료로 마무리한다.
type PaymentRefunded struct {
	OrderID OrderID `json:"order_id"`
	Amount  int64   `json:"amount"`
}

func (PaymentRefunded) EventName() string { return "payment.refunded" }
