package app

import (
	"context"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// --- 테스트용 fake 들 ---

type fakeStockRepo struct {
	items map[domain.ProductID]*domain.StockItem
}

func newFakeStockRepo(seed map[string]int) *fakeStockRepo {
	items := map[domain.ProductID]*domain.StockItem{}
	for id, n := range seed {
		items[domain.ProductID(id)] = domain.NewStockItem(domain.ProductID(id), n)
	}
	return &fakeStockRepo{items: items}
}

func (r *fakeStockRepo) FindByProduct(_ context.Context, id domain.ProductID) (*domain.StockItem, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, domain.ErrStockItemNotFound
	}
	return item, nil
}
func (r *fakeStockRepo) Update(_ context.Context, id domain.ProductID, mutate func(*domain.StockItem) error) error {
	item, ok := r.items[id]
	if !ok {
		return domain.ErrStockItemNotFound
	}
	return mutate(item)
}

type fakePublisher struct{ published []domain.DomainEvent }

func (p *fakePublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	p.published = append(p.published, events...)
	return nil
}

type fakeReservationRepo struct {
	store map[domain.OrderID]*domain.Reservation
}

func newFakeReservationRepo() *fakeReservationRepo {
	return &fakeReservationRepo{store: map[domain.OrderID]*domain.Reservation{}}
}
func (r *fakeReservationRepo) Save(_ context.Context, res *domain.Reservation) error {
	r.store[res.OrderID()] = res
	return nil
}
func (r *fakeReservationRepo) Find(_ context.Context, id domain.OrderID) (*domain.Reservation, error) {
	res, ok := r.store[id]
	if !ok {
		return nil, domain.ErrReservationNotFound
	}
	return res, nil
}
func (r *fakeReservationRepo) Delete(_ context.Context, id domain.OrderID) error {
	delete(r.store, id)
	return nil
}

func newService(seed map[string]int) (*ReservationService, *fakeStockRepo, *fakePublisher) {
	repo := newFakeStockRepo(seed)
	pub := &fakePublisher{}
	return NewReservationService(repo, newFakeReservationRepo(), pub), repo, pub
}

func cmd() ReserveForOrderCommand {
	return ReserveForOrderCommand{
		OrderID: "order-1",
		Items: []ReservationItem{
			{ProductID: "prod-A", Quantity: 2},
			{ProductID: "prod-B", Quantity: 1},
		},
	}
}

func TestOnOrderPlaced_재고가_충분하면_예약되고_이벤트발행(t *testing.T) {
	svc, repo, pub := newService(map[string]int{"prod-A": 10, "prod-B": 5})

	if err := svc.OnOrderPlaced(context.Background(), cmd()); err != nil {
		t.Fatalf("OnOrderPlaced: %v", err)
	}

	// 재고가 차감됐는가
	a, _ := repo.FindByProduct(context.Background(), "prod-A")
	b, _ := repo.FindByProduct(context.Background(), "prod-B")
	if a.Available() != 8 || b.Available() != 4 {
		t.Fatalf("예약 후 재고 A=8,B=4 여야 하는데 A=%d,B=%d", a.Available(), b.Available())
	}
	// StockReserved 가 발행됐는가
	if len(pub.published) != 1 {
		t.Fatalf("이벤트 1개 발행돼야 하는데 %d개", len(pub.published))
	}
	if _, ok := pub.published[0].(domain.StockReserved); !ok {
		t.Fatalf("발행 이벤트는 StockReserved 여야 함: %T", pub.published[0])
	}
}

func TestOnOrderPlaced_한_항목이라도_부족하면_보상하고_실패발행(t *testing.T) {
	// prod-A 는 넉넉하지만 prod-B 가 부족하다.
	svc, repo, pub := newService(map[string]int{"prod-A": 10, "prod-B": 0})

	if err := svc.OnOrderPlaced(context.Background(), cmd()); err != nil {
		t.Fatalf("OnOrderPlaced 는 비즈니스 실패를 이벤트로 다뤄야 하므로 nil 이어야: %v", err)
	}

	// 먼저 예약했던 prod-A 는 보상으로 원상복구(10)돼야 한다.
	a, _ := repo.FindByProduct(context.Background(), "prod-A")
	if a.Available() != 10 {
		t.Fatalf("보상 후 prod-A 재고 = 10 이어야 하는데 %d", a.Available())
	}
	// 실패 이벤트가 발행됐는가
	if len(pub.published) != 1 {
		t.Fatalf("이벤트 1개 발행돼야 하는데 %d개", len(pub.published))
	}
	if _, ok := pub.published[0].(domain.StockReservationFailed); !ok {
		t.Fatalf("발행 이벤트는 StockReservationFailed 여야 함: %T", pub.published[0])
	}
}

func TestOnOrderCancelled_예약재고가_복원된다(t *testing.T) {
	svc, repo, _ := newService(map[string]int{"prod-A": 10, "prod-B": 5})

	// 먼저 예약(A 2개·B 1개) → 재고 A=8, B=4
	if err := svc.OnOrderPlaced(context.Background(), cmd()); err != nil {
		t.Fatalf("OnOrderPlaced: %v", err)
	}
	// 주문 취소 → 예약 복원
	if err := svc.OnOrderCancelled(context.Background(), "order-1"); err != nil {
		t.Fatalf("OnOrderCancelled: %v", err)
	}

	a, _ := repo.FindByProduct(context.Background(), "prod-A")
	b, _ := repo.FindByProduct(context.Background(), "prod-B")
	if a.Available() != 10 || b.Available() != 5 {
		t.Fatalf("복원 후 재고 A=10,B=5 여야 하는데 A=%d,B=%d", a.Available(), b.Available())
	}

	// 예약 기록이 지워져, 중복 취소가 와도 두 번 복원하지 않는다(멱등).
	if err := svc.OnOrderCancelled(context.Background(), "order-1"); err != nil {
		t.Fatalf("중복 취소: %v", err)
	}
	a2, _ := repo.FindByProduct(context.Background(), "prod-A")
	if a2.Available() != 10 {
		t.Fatalf("중복 취소로 재고가 늘면 안 되는데 A=%d", a2.Available())
	}
}
