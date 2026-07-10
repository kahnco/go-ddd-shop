package app

import (
	"context"
	"errors"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/cart/domain"
)

// 테스트용 fake 저장소(infra 를 import 하면 순환이 되므로 여기서 만든다).
type fakeCartRepo struct{ store map[string][]domain.CartItem }

func newFakeCartRepo() *fakeCartRepo { return &fakeCartRepo{store: map[string][]domain.CartItem{}} }
func (r *fakeCartRepo) Save(_ context.Context, c *domain.Cart) error {
	r.store[c.CustomerID()] = c.Items()
	return nil
}
func (r *fakeCartRepo) Find(_ context.Context, id string) (*domain.Cart, error) {
	items, ok := r.store[id]
	if !ok {
		return nil, domain.ErrCartNotFound
	}
	return domain.Load(id, items), nil
}
func (r *fakeCartRepo) Delete(_ context.Context, id string) error {
	delete(r.store, id)
	return nil
}

type fakeCustomers struct{ known map[string]bool }

func (c fakeCustomers) Exists(_ context.Context, id string) (bool, error) { return c.known[id], nil }

type fakeOrders struct {
	placedCustomer string
	placedItems    []OrderItem
}

func (o *fakeOrders) Place(_ context.Context, customerID string, items []OrderItem) (string, error) {
	o.placedCustomer = customerID
	o.placedItems = items
	return "order-1", nil
}

func newService(known map[string]bool) (*CartService, *fakeCartRepo, *fakeOrders) {
	carts := newFakeCartRepo()
	orders := &fakeOrders{}
	return NewCartService(carts, fakeCustomers{known: known}, orders), carts, orders
}

func TestCheckout_회원이면_주문이_생성되고_장바구니가_비워진다(t *testing.T) {
	svc, carts, orders := newService(map[string]bool{"cust-1": true})
	ctx := context.Background()
	_, _ = svc.AddItem(ctx, "cust-1", "prod-A", 2)
	_, _ = svc.AddItem(ctx, "cust-1", "prod-B", 1)

	orderID, err := svc.Checkout(ctx, "cust-1")
	if err != nil {
		t.Fatalf("Checkout: %v", err)
	}
	if orderID != "order-1" {
		t.Fatalf("주문 ID = order-1 여야 하는데 %s", orderID)
	}
	if len(orders.placedItems) != 2 {
		t.Fatalf("주문에 항목 2개 실려야 하는데 %d개", len(orders.placedItems))
	}
	// 결제 후 장바구니는 비워져야 한다.
	if _, err := carts.Find(ctx, "cust-1"); !errors.Is(err, domain.ErrCartNotFound) {
		t.Fatalf("결제 후 장바구니는 비워져야 하는데: %v", err)
	}
}

func TestCheckout_비회원은_거부된다(t *testing.T) {
	svc, _, _ := newService(map[string]bool{}) // 아무도 회원이 아님
	_, _ = svc.AddItem(context.Background(), "guest", "prod-A", 1)

	if _, err := svc.Checkout(context.Background(), "guest"); !errors.Is(err, ErrCustomerNotRegistered) {
		t.Fatalf("비회원 결제는 ErrCustomerNotRegistered 여야 하는데: %v", err)
	}
}

func TestCheckout_빈_장바구니는_거부된다(t *testing.T) {
	svc, _, _ := newService(map[string]bool{"cust-1": true})
	if _, err := svc.Checkout(context.Background(), "cust-1"); !errors.Is(err, domain.ErrEmptyCart) {
		t.Fatalf("빈 장바구니 결제는 ErrEmptyCart 여야 하는데: %v", err)
	}
}
