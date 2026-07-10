package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/kahnco/go-ddd-shop/internal/shipping/domain"
)

// EventPublisher 는 배송 컨텍스트가 자신의 이벤트를 발행하는 포트.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}

// DispatchCommand 는 "이 주문이 확정됐으니 배송하라"는 입력.
// 주문 컨텍스트의 OrderConfirmed 이벤트에서 번역돼 들어온다.
type DispatchCommand struct {
	OrderID string
}

// ShippingService 는 배송 유스케이스를 담는 애플리케이션 서비스.
type ShippingService struct {
	repo      domain.ShipmentRepository
	publisher EventPublisher
}

func NewShippingService(repo domain.ShipmentRepository, publisher EventPublisher) *ShippingService {
	return &ShippingService{repo: repo, publisher: publisher}
}

// OnOrderConfirmed 는 주문 확정에 반응해 배송을 시작하고 ShipmentDispatched 를 발행한다.
func (s *ShippingService) OnOrderConfirmed(ctx context.Context, cmd DispatchCommand) error {
	shipment := domain.NewShipment(domain.OrderID(cmd.OrderID))
	shipment.Dispatch(newTrackingNo())

	if err := s.repo.Save(ctx, shipment); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, shipment.PullEvents()...)
}

func newTrackingNo() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "TRK-" + hex.EncodeToString(b)
}
