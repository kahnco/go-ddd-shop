package app

import (
	"context"
	"errors"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// 테스트용 가짜(fake) 구현들. 포트가 인터페이스라 이렇게 갈아끼울 수 있다.

type fakeRepo struct {
	saved map[domain.OrderID]*domain.Order
}

func newFakeRepo() *fakeRepo { return &fakeRepo{saved: map[domain.OrderID]*domain.Order{}} }

func (r *fakeRepo) Save(_ context.Context, o *domain.Order) error {
	r.saved[o.ID()] = o
	return nil
}
func (r *fakeRepo) FindByID(_ context.Context, id domain.OrderID) (*domain.Order, error) {
	o, ok := r.saved[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

type fakePublisher struct {
	published []domain.DomainEvent
}

func (p *fakePublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	p.published = append(p.published, events...)
	return nil
}

// 고정 ID 를 반환해 결과를 결정적으로 검증.
type fixedID struct{ id domain.OrderID }

func (f fixedID) NewOrderID() domain.OrderID { return f.id }

// 가격 조회를 흉내 내는 fake. 카탈로그 프로젝션 자리를 대신한다.
type fakePrices struct{ m map[domain.ProductID]int64 }

func (f fakePrices) PriceOf(_ context.Context, id domain.ProductID) (domain.Money, error) {
	p, ok := f.m[id]
	if !ok {
		return domain.Money{}, domain.ErrUnknownProduct
	}
	return domain.NewMoney(p)
}

func newService() (*OrderService, *fakeRepo, *fakePublisher) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	prices := fakePrices{m: map[domain.ProductID]int64{"prod-A": 1000, "prod-B": 3000}}
	svc := NewOrderService(repo, pub, fixedID{id: "order-fixed"}, prices)
	return svc, repo, pub
}

func validCommand() PlaceOrderCommand {
	return PlaceOrderCommand{
		CustomerID: "cust-1",
		Items: []OrderItemInput{
			{ProductID: "prod-A", Quantity: 2}, // 가격은 카탈로그에서(prod-A=1000)
			{ProductID: "prod-B", Quantity: 1}, // prod-B=3000
		},
	}
}

func TestPlaceOrder_성공하면_저장되고_이벤트가_발행된다(t *testing.T) {
	svc, repo, pub := newService()

	id, err := svc.PlaceOrder(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if id != "order-fixed" {
		t.Fatalf("반환 ID = order-fixed 여야 하는데 %s", id)
	}

	// 저장됐는가
	saved, err := repo.FindByID(context.Background(), id)
	if err != nil {
		t.Fatalf("저장된 주문을 찾을 수 없음: %v", err)
	}
	if saved.Total().Amount() != 5000 {
		t.Fatalf("저장된 총액 = 5000 이어야 하는데 %d", saved.Total().Amount())
	}

	// OrderPlaced 이벤트가 발행됐는가
	if len(pub.published) != 1 {
		t.Fatalf("이벤트 1개 발행돼야 하는데 %d개", len(pub.published))
	}
	if _, ok := pub.published[0].(domain.OrderPlaced); !ok {
		t.Fatalf("발행된 이벤트는 OrderPlaced 여야 함: %T", pub.published[0])
	}
}

func TestPlaceOrder_잘못된_수량이면_에러이고_아무것도_저장안됨(t *testing.T) {
	svc, repo, pub := newService()

	cmd := validCommand()
	cmd.Items[0].Quantity = 0 // 도메인 규칙 위반

	_, err := svc.PlaceOrder(context.Background(), cmd)
	if !errors.Is(err, domain.ErrNonPositiveQuantity) {
		t.Fatalf("수량 0 은 ErrNonPositiveQuantity 여야 하는데: %v", err)
	}
	// 실패했으면 저장도, 발행도 없어야 한다
	if len(repo.saved) != 0 {
		t.Fatalf("실패 시 저장이 없어야 하는데 %d개", len(repo.saved))
	}
	if len(pub.published) != 0 {
		t.Fatalf("실패 시 발행이 없어야 하는데 %d개", len(pub.published))
	}
}

func TestPlaceOrder_카탈로그에_없는_상품이면_거부된다(t *testing.T) {
	svc, repo, _ := newService()

	cmd := validCommand()
	cmd.Items[0].ProductID = "prod-없음" // 가격 프로젝션에 없는 상품

	_, err := svc.PlaceOrder(context.Background(), cmd)
	if !errors.Is(err, domain.ErrUnknownProduct) {
		t.Fatalf("모르는 상품은 ErrUnknownProduct 여야 하는데: %v", err)
	}
	if len(repo.saved) != 0 {
		t.Fatalf("실패 시 저장이 없어야 하는데 %d개", len(repo.saved))
	}
}
