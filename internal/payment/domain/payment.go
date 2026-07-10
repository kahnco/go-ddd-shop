// Package domain 은 결제(payment) bounded context 의 도메인이다.
// 다른 컨텍스트와 코드를 공유하지 않고, 이벤트 계약(JSON)으로만 연결된다.
package domain

// OrderID 는 결제 컨텍스트가 보는 주문 식별자. 다른 컨텍스트의 동명 타입과는 별개다.
type OrderID string

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "PENDING"
	StatusCompleted PaymentStatus = "COMPLETED"
	StatusFailed    PaymentStatus = "FAILED"
)

// Payment 는 한 주문에 대한 결제. 애그리거트 루트로서 결제 시도의 결과를 스스로 정한다.
type Payment struct {
	orderID OrderID
	amount  int64
	status  PaymentStatus
	events  []DomainEvent
}

func NewPayment(orderID OrderID, amount int64) *Payment {
	return &Payment{orderID: orderID, amount: amount, status: StatusPending}
}

// autoApproveLimit 는 데모용 자동 승인 한도(원). 이보다 큰 결제는 "거절"로 흉내 낸다.
// 실제라면 이 자리에서 PG 게이트웨이의 승인/거절 응답을 받는다.
const autoApproveLimit int64 = 1_000_000

// Process 는 결제를 시도한다. 지금은 데모용 목업 —
// 금액이 0 이하이거나 한도를 넘으면 거절, 그 외엔 승인한다.
// 결과는 상태로 남고, 그에 맞는 도메인 이벤트를 기록한다.
func (p *Payment) Process() {
	switch {
	case p.amount <= 0:
		p.decline("유효하지 않은 결제 금액")
	case p.amount > autoApproveLimit:
		p.decline("한도 초과로 결제가 거절되었습니다")
	default:
		p.status = StatusCompleted
		p.record(PaymentCompleted{OrderID: p.orderID, Amount: p.amount})
	}
}

func (p *Payment) decline(reason string) {
	p.status = StatusFailed
	p.record(PaymentFailed{OrderID: p.orderID, Reason: reason})
}

func (p *Payment) OrderID() OrderID      { return p.orderID }
func (p *Payment) Status() PaymentStatus { return p.status }
func (p *Payment) record(e DomainEvent)  { p.events = append(p.events, e) }
func (p *Payment) PullEvents() []DomainEvent {
	e := p.events
	p.events = nil
	return e
}
