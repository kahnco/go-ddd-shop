package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// MemoryReservationRepository 는 ReservationRepository 포트를 메모리로 구현한 어댑터.
type MemoryReservationRepository struct {
	mu    sync.Mutex
	store map[domain.OrderID]*domain.Reservation
}

func NewMemoryReservationRepository() *MemoryReservationRepository {
	return &MemoryReservationRepository{store: make(map[domain.OrderID]*domain.Reservation)}
}

func (r *MemoryReservationRepository) Save(_ context.Context, reservation *domain.Reservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[reservation.OrderID()] = reservation
	return nil
}

func (r *MemoryReservationRepository) Find(_ context.Context, orderID domain.OrderID) (*domain.Reservation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.store[orderID]
	if !ok {
		return nil, domain.ErrReservationNotFound
	}
	return res, nil
}

func (r *MemoryReservationRepository) Delete(_ context.Context, orderID domain.OrderID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, orderID)
	return nil
}
