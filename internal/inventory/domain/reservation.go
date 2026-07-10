package domain

// Reservation 은 "어떤 주문을 위해 무엇을 얼마나 예약했는지"의 기록이다.
// 결제 실패 등으로 주문이 취소될 때, 이 기록을 보고 예약을 정확히 되돌린다(보상).
// 이 기록이 없으면 "무엇을 풀어야 하는지" 알 수 없다.
type Reservation struct {
	orderID OrderID
	items   []ReservedItem
}

type ReservedItem struct {
	ProductID ProductID
	Quantity  int
}

func NewReservation(orderID OrderID) *Reservation {
	return &Reservation{orderID: orderID}
}

func (r *Reservation) Add(productID ProductID, quantity int) {
	r.items = append(r.items, ReservedItem{ProductID: productID, Quantity: quantity})
}

func (r *Reservation) OrderID() OrderID      { return r.orderID }
func (r *Reservation) Items() []ReservedItem { return r.items }
