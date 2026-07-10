package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
)

// MemoryCustomerRepository 는 CustomerRepository 포트를 메모리로 구현한 어댑터.
type MemoryCustomerRepository struct {
	mu    sync.RWMutex
	store map[domain.CustomerID]*domain.Customer
}

func NewMemoryCustomerRepository() *MemoryCustomerRepository {
	return &MemoryCustomerRepository{store: make(map[domain.CustomerID]*domain.Customer)}
}

func (r *MemoryCustomerRepository) Save(_ context.Context, c *domain.Customer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[c.ID()] = c
	return nil
}

func (r *MemoryCustomerRepository) Find(_ context.Context, id domain.CustomerID) (*domain.Customer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.store[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return c, nil
}
