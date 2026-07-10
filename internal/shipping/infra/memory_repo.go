package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/shipping/domain"
)

// MemoryShipmentRepository 는 ShipmentRepository 포트를 메모리로 구현한 어댑터.
type MemoryShipmentRepository struct {
	mu    sync.Mutex
	store map[domain.OrderID]*domain.Shipment
}

func NewMemoryShipmentRepository() *MemoryShipmentRepository {
	return &MemoryShipmentRepository{store: make(map[domain.OrderID]*domain.Shipment)}
}

func (r *MemoryShipmentRepository) Save(_ context.Context, shipment *domain.Shipment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[shipment.OrderID()] = shipment
	return nil
}
