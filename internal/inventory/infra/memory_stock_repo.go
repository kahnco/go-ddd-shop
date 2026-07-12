package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// MemoryStockRepository 는 StockRepository 포트를 메모리로 구현한 어댑터.
// 모든 수정을 하나의 뮤텍스 아래 Update 로 처리해, 동시 접근에도 원자적이다.
type MemoryStockRepository struct {
	mu    sync.Mutex
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

// FindByProduct 는 재고의 스냅샷(복사본)을 돌려준다.
// 복사본이라 바깥에서 이걸 바꿔도 저장소엔 반영되지 않는다 — 수정은 오직 Update 로.
func (r *MemoryStockRepository) FindByProduct(_ context.Context, id domain.ProductID) (*domain.StockItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.store[id]
	if !ok {
		return nil, domain.ErrStockItemNotFound
	}
	snapshot := *item
	return &snapshot, nil
}

// Update 는 조회·수정·저장을 한 락 안에서 원자적으로 처리한다.
// mutate 안의 read-modify-write(Reserve/Release/Restock)가 다른 요청과 겹치지 않는다.
func (r *MemoryStockRepository) Update(_ context.Context, id domain.ProductID, mutate func(*domain.StockItem) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.store[id]
	if !ok {
		return domain.ErrStockItemNotFound
	}
	return mutate(item)
}
