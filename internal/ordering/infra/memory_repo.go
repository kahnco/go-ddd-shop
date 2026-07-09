package infra

import (
	"context"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// MemoryOrderRepository 는 도메인의 OrderRepository 포트를 메모리로 구현한 어댑터.
// 3편에서는 이걸 쓰고, 뒤에서 같은 인터페이스를 PostgreSQL 어댑터로 갈아끼운다.
// 이렇게 도메인·애플리케이션 코드를 하나도 안 바꾸고 저장소만 교체할 수 있다는 게
// 포트/어댑터(헥사고날)의 핵심 이득이다.
type MemoryOrderRepository struct {
	mu    sync.RWMutex
	store map[domain.OrderID]*domain.Order
}

func NewMemoryOrderRepository() *MemoryOrderRepository {
	return &MemoryOrderRepository{store: make(map[domain.OrderID]*domain.Order)}
}

func (r *MemoryOrderRepository) Save(_ context.Context, order *domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[order.ID()] = order
	return nil
}

func (r *MemoryOrderRepository) FindByID(_ context.Context, id domain.OrderID) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.store[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return order, nil
}
