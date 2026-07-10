package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/cart/domain"
)

// MemoryCartRepository 는 CartRepository 포트를 메모리로 구현한 어댑터.
type MemoryCartRepository struct {
	mu    sync.Mutex
	store map[string][]domain.CartItem
}

func NewMemoryCartRepository() *MemoryCartRepository {
	return &MemoryCartRepository{store: make(map[string][]domain.CartItem)}
}

func (r *MemoryCartRepository) Save(_ context.Context, cart *domain.Cart) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[cart.CustomerID()] = cart.Items()
	return nil
}

func (r *MemoryCartRepository) Find(_ context.Context, customerID string) (*domain.Cart, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	items, ok := r.store[customerID]
	if !ok {
		return nil, domain.ErrCartNotFound
	}
	return domain.Load(customerID, items), nil
}

func (r *MemoryCartRepository) Delete(_ context.Context, customerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.store, customerID)
	return nil
}
