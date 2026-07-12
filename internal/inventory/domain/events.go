package domain

// 재고 컨텍스트가 발행하는 도메인 이벤트. 다른 컨텍스트는 이 JSON 계약을 구독한다.

type DomainEvent interface {
	EventName() string
}

// StockReserved — 주문의 모든 항목 재고가 예약됨. 결제 컨텍스트가 이어받아 결제한다.
// 결제가 청구할 금액(Amount)을 실어 보낸다 — 결제는 주문을 되묻지 않아도 된다.
type StockReserved struct {
	OrderID OrderID `json:"order_id"`
	Amount  int64   `json:"amount"`
}

func (StockReserved) EventName() string { return "stock.reserved" }

// StockReservationFailed — 재고 부족 등으로 예약 실패.
// 주문 컨텍스트가 구독해 주문을 취소한다(보상). 사가(saga)의 실패 경로다.
type StockReservationFailed struct {
	OrderID OrderID `json:"order_id"`
	Reason  string  `json:"reason"`
}

func (StockReservationFailed) EventName() string { return "stock.reservation_failed" }

// StockRestocked — 반품으로 재고가 다시 채워짐.
type StockRestocked struct {
	OrderID OrderID `json:"order_id"`
}

func (StockRestocked) EventName() string { return "stock.restocked" }
