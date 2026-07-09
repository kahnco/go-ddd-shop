package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// MemoryStockRepository 는 StockRepository 포트를 메모리로 구현한 어댑터.
type MemoryStockRepository struct {
	mu    sync.RWMutex
	store map[domain.ProductID]*domain.StockItem
}

func NewMemoryStockRepository() *MemoryStockRepository {
	return &MemoryStockRepository{store: make(map[domain.ProductID]*domain.StockItem)}
}

// Seed 는 초기 재고를 채운다(데모·테스트용).
func (r *MemoryStockRepository) Seed(id domain.ProductID, available int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[id] = domain.NewStockItem(id, available)
}

func (r *MemoryStockRepository) FindByProduct(_ context.Context, id domain.ProductID) (*domain.StockItem, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.store[id]
	if !ok {
		return nil, domain.ErrStockItemNotFound
	}
	return item, nil
}

func (r *MemoryStockRepository) Save(_ context.Context, item *domain.StockItem) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[item.ProductID()] = item
	return nil
}
