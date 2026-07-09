package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/payment/domain"
)

// MemoryPaymentRepository 는 PaymentRepository 포트를 메모리로 구현한 어댑터.
type MemoryPaymentRepository struct {
	mu    sync.Mutex
	store map[domain.OrderID]*domain.Payment
}

func NewMemoryPaymentRepository() *MemoryPaymentRepository {
	return &MemoryPaymentRepository{store: make(map[domain.OrderID]*domain.Payment)}
}

func (r *MemoryPaymentRepository) Save(_ context.Context, payment *domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[payment.OrderID()] = payment
	return nil
}
