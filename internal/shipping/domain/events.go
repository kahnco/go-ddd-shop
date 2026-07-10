package domain

type DomainEvent interface {
	EventName() string
}

// ShipmentDispatched — 배송 시작됨. 주문 컨텍스트가 구독해 주문을 배송중으로 전이한다.
type ShipmentDispatched struct {
	OrderID    OrderID `json:"order_id"`
	TrackingNo string  `json:"tracking_no"`
}

func (ShipmentDispatched) EventName() string { return "shipping.dispatched" }
