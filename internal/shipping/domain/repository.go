package domain

import "context"

// ShipmentRepository 는 배송 저장소 포트. 구현은 인프라 계층이 채운다.
type ShipmentRepository interface {
	Save(ctx context.Context, shipment *Shipment) error
}
