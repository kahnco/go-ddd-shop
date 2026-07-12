package domain

import "encoding/json"

// 값 객체(Value Object)들.
// 값 객체는 식별자가 없고 값 자체로 동등하며, 한번 만들어지면 바뀌지 않는다(불변).
// 생성 시점에 유효성을 강제해, 이후 코드에서는 "항상 유효한 값"이라고 믿을 수 있게 한다.

// Money 는 금액(원, KRW). 음수가 될 수 없다.
type Money struct {
	amount int64
}

// NewMoney 는 유효한 금액만 만든다. 음수면 에러.
func NewMoney(amount int64) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeMoney
	}
	return Money{amount: amount}, nil
}

func (m Money) Amount() int64       { return m.amount }
func (m Money) Add(o Money) Money   { return Money{amount: m.amount + o.amount} }
func (m Money) Equals(o Money) bool { return m.amount == o.amount }

// MarshalJSON — 금액은 겉으로는 그냥 정수(원)다. 이벤트를 브로커로 내보낼 때
// Money 가 스칼라 값으로 직렬화되도록 한다(내부 필드가 비공개라 기본 직렬화로는 {} 가 된다).
func (m Money) MarshalJSON() ([]byte, error) { return json.Marshal(m.amount) }

// Times 는 금액에 수량을 곱한다(주문 항목 소계 계산용).
func (m Money) Times(q Quantity) Money {
	return Money{amount: m.amount * int64(q.value)}
}

// Quantity 는 주문 수량. 1 이상이어야 한다.
type Quantity struct {
	value int
}

func NewQuantity(v int) (Quantity, error) {
	if v <= 0 {
		return Quantity{}, ErrNonPositiveQuantity
	}
	return Quantity{value: v}, nil
}

func (q Quantity) Value() int { return q.value }

// 식별자들. 도메인 언어를 타입으로 드러내, string 끼리 섞이는 실수를 막는다.
type (
	OrderID    string
	ProductID  string
	CustomerID string
)

// OrderStatus 는 주문의 상태. 정해진 순서로만 전이된다(order.go 의 allowedTransitions).
type OrderStatus string

const (
	StatusPlaced          OrderStatus = "PLACED"           // 생성됨
	StatusPaid            OrderStatus = "PAID"             // 결제완료
	StatusConfirmed       OrderStatus = "CONFIRMED"        // 확정
	StatusShipped         OrderStatus = "SHIPPED"          // 배송중
	StatusCancelled       OrderStatus = "CANCELLED"        // 취소
	StatusReturnRequested OrderStatus = "RETURN_REQUESTED" // 반품 요청됨
	StatusRefunded        OrderStatus = "REFUNDED"         // 환불 완료
)
