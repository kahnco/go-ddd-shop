// Package domain 은 배송(shipping) bounded context 의 도메인이다.
package domain

type OrderID string

type ShipmentStatus string

const (
	StatusPending    ShipmentStatus = "PENDING"
	StatusDispatched ShipmentStatus = "DISPATCHED"
)

// Shipment 는 한 주문의 배송. 애그리거트 루트.
type Shipment struct {
	orderID    OrderID
	trackingNo string
	status     ShipmentStatus
	events     []DomainEvent
}

func NewShipment(orderID OrderID) *Shipment {
	return &Shipment{orderID: orderID, status: StatusPending}
}

// Dispatch 는 배송을 시작한다(운송장 번호 부여). 데모용 목업 —
// 실제라면 택배사 API 를 호출해 운송장을 받는 자리다.
func (s *Shipment) Dispatch(trackingNo string) {
	s.trackingNo = trackingNo
	s.status = StatusDispatched
	s.record(ShipmentDispatched{OrderID: s.orderID, TrackingNo: trackingNo})
}

func (s *Shipment) OrderID() OrderID       { return s.orderID }
func (s *Shipment) TrackingNo() string     { return s.trackingNo }
func (s *Shipment) Status() ShipmentStatus { return s.status }
func (s *Shipment) record(e DomainEvent)   { s.events = append(s.events, e) }
func (s *Shipment) PullEvents() []DomainEvent {
	e := s.events
	s.events = nil
	return e
}
