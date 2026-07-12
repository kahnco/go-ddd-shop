package domain

import "fmt"

// OrderLine 은 주문 안의 "어떤 상품 몇 개를 얼마에". Order 애그리거트의 일부다.
type OrderLine struct {
	productID ProductID
	quantity  Quantity
	unitPrice Money
}

func NewOrderLine(productID ProductID, quantity Quantity, unitPrice Money) OrderLine {
	return OrderLine{productID: productID, quantity: quantity, unitPrice: unitPrice}
}

// Subtotal 은 이 항목의 소계 = 단가 × 수량.
func (l OrderLine) Subtotal() Money      { return l.unitPrice.Times(l.quantity) }
func (l OrderLine) ProductID() ProductID { return l.productID }
func (l OrderLine) Quantity() Quantity   { return l.quantity }
func (l OrderLine) UnitPrice() Money     { return l.unitPrice }

// Order 는 주문 애그리거트. 애그리거트 루트로서 OrderLine 들을 품고,
// 불변식(항목 1개 이상, 총액=소계 합, 정해진 상태 전이)을 스스로 지킨다.
// 바깥은 반드시 이 루트를 통해서만 주문을 조작한다.
type Order struct {
	id         OrderID
	customerID CustomerID
	lines      []OrderLine
	status     OrderStatus
	events     []DomainEvent
}

// PlaceOrder 는 새 주문을 생성하는 팩토리. 불변식을 여기서 강제한다.
// 항목이 하나도 없으면 애초에 주문이 만들어지지 않는다.
func PlaceOrder(id OrderID, customerID CustomerID, lines []OrderLine) (*Order, error) {
	if len(lines) == 0 {
		return nil, ErrEmptyOrder
	}
	o := &Order{
		id:         id,
		customerID: customerID,
		lines:      lines,
		status:     StatusPlaced,
	}
	// 이벤트에 항목을 실어 보낸다. 재고 컨텍스트가 이 데이터만으로 예약할 수 있게.
	items := make([]OrderPlacedItem, len(lines))
	for i, l := range lines {
		items[i] = OrderPlacedItem{ProductID: l.productID, Quantity: l.quantity.value}
	}
	o.record(OrderPlaced{OrderID: id, CustomerID: customerID, Total: o.Total(), Items: items})
	return o, nil
}

// ReconstituteOrder 는 저장소가 "이미 존재하는" 주문을 메모리로 되살릴 때 쓴다.
// PlaceOrder(팩토리)와 결정적으로 다른 점: 이벤트를 하나도 발생시키지 않는다.
// 이건 지금 새로 일어나는 일이 아니라, 과거에 저장된 사실을 복원하는 것이기 때문이다.
// (이벤트를 또 내면 재고가 다시 예약되는 등 끔찍한 중복이 생긴다.)
func ReconstituteOrder(id OrderID, customerID CustomerID, lines []OrderLine, status OrderStatus) *Order {
	return &Order{id: id, customerID: customerID, lines: lines, status: status}
}

// Total 은 모든 항목 소계의 합. 저장된 값이 아니라 항상 항목에서 계산해,
// "총액 = 소계 합" 불변식이 절대 깨지지 않게 한다.
func (o *Order) Total() Money {
	total := Money{}
	for _, l := range o.lines {
		total = total.Add(l.Subtotal())
	}
	return total
}

// 허용된 상태 전이. 여기 없는 전이는 모두 거부된다.
var allowedTransitions = map[OrderStatus][]OrderStatus{
	StatusPlaced:          {StatusPaid, StatusCancelled},
	StatusPaid:            {StatusConfirmed, StatusCancelled},
	StatusConfirmed:       {StatusShipped},
	StatusShipped:         {StatusReturnRequested}, // 배송된 주문만 반품 가능
	StatusReturnRequested: {StatusRefunded},        // 반품 요청 → 환불 완료
}

func (o *Order) transition(to OrderStatus, event DomainEvent) error {
	for _, allowed := range allowedTransitions[o.status] {
		if allowed == to {
			o.status = to
			o.record(event)
			return nil
		}
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidStatusTransition, o.status, to)
}

func (o *Order) MarkPaid() error { return o.transition(StatusPaid, OrderPaid{OrderID: o.id}) }
func (o *Order) Confirm() error  { return o.transition(StatusConfirmed, OrderConfirmed{OrderID: o.id}) }
func (o *Order) Ship() error     { return o.transition(StatusShipped, OrderShipped{OrderID: o.id}) }
func (o *Order) Cancel() error   { return o.transition(StatusCancelled, OrderCancelled{OrderID: o.id}) }

// RequestReturn 은 배송된 주문의 반품을 요청한다(사후 보상 사가의 시작).
// 이벤트에 항목·금액을 실어, 결제는 환불하고 재고는 복원할 수 있게 한다.
func (o *Order) RequestReturn() error {
	items := make([]OrderPlacedItem, len(o.lines))
	for i, l := range o.lines {
		items[i] = OrderPlacedItem{ProductID: l.productID, Quantity: l.quantity.value}
	}
	return o.transition(StatusReturnRequested, OrderReturnRequested{
		OrderID: o.id, Amount: o.Total(), Items: items,
	})
}

// MarkRefunded 는 환불 완료에 반응해 주문을 환불완료로 전이한다.
func (o *Order) MarkRefunded() error {
	return o.transition(StatusRefunded, OrderRefunded{OrderID: o.id})
}

func (o *Order) ID() OrderID            { return o.id }
func (o *Order) CustomerID() CustomerID { return o.customerID }
func (o *Order) Status() OrderStatus    { return o.status }

// Lines 는 내부 슬라이스를 복사해 돌려준다. 바깥에서 항목을 몰래 바꿔
// 불변식을 깨는 것을 막기 위해서다(애그리거트 캡슐화).
func (o *Order) Lines() []OrderLine {
	out := make([]OrderLine, len(o.lines))
	copy(out, o.lines)
	return out
}

func (o *Order) record(e DomainEvent) { o.events = append(o.events, e) }

// PullEvents 는 그동안 쌓인 도메인 이벤트를 꺼내고 비운다.
// 애플리케이션 계층이 이걸 꺼내 밖으로 발행한다(4편에서 브로커로).
func (o *Order) PullEvents() []DomainEvent {
	e := o.events
	o.events = nil
	return e
}
