package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
)

// MemoryProductRepository 는 ProductRepository 포트를 메모리로 구현한 어댑터.
type MemoryProductRepository struct {
	mu    sync.RWMutex
	store map[domain.ProductID]*domain.Product
}

func NewMemoryProductRepository() *MemoryProductRepository {
	return &MemoryProductRepository{store: make(map[domain.ProductID]*domain.Product)}
}

func (r *MemoryProductRepository) Save(_ context.Context, p *domain.Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[p.ID()] = p
	return nil
}

func (r *MemoryProductRepository) Find(_ context.Context, id domain.ProductID) (*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.store[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (r *MemoryProductRepository) All(_ context.Context) ([]*domain.Product, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Product, 0, len(r.store))
	for _, p := range r.store {
		out = append(out, p)
	}
	return out, nil
}
