package app

import (
	"context"
	"errors"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
)

type fakeRepo struct {
	store map[domain.CustomerID]*domain.Customer
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[domain.CustomerID]*domain.Customer{}} }
func (r *fakeRepo) Save(_ context.Context, c *domain.Customer) error {
	r.store[c.ID()] = c
	return nil
}
func (r *fakeRepo) Find(_ context.Context, id domain.CustomerID) (*domain.Customer, error) {
	c, ok := r.store[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return c, nil
}

type fakePublisher struct{ published []domain.DomainEvent }

func (p *fakePublisher) Publish(_ context.Context, e ...domain.DomainEvent) error {
	p.published = append(p.published, e...)
	return nil
}

func TestRegister_등록되고_이벤트발행(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	svc := NewCustomerService(repo, pub)

	if err := svc.Register(context.Background(), "cust-1", "a@b.com", "홍길동"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, ok := pub.published[0].(domain.CustomerRegistered); !ok {
		t.Fatalf("CustomerRegistered 발행돼야 함: %T", pub.published[0])
	}
	got, _ := svc.Get(context.Background(), "cust-1")
	if got.Email() != "a@b.com" {
		t.Fatalf("이메일 = a@b.com 여야 하는데 %s", got.Email())
	}
}

func TestRegister_중복은_거부된다(t *testing.T) {
	svc := NewCustomerService(newFakeRepo(), &fakePublisher{})
	_ = svc.Register(context.Background(), "cust-1", "a@b.com", "홍길동")

	err := svc.Register(context.Background(), "cust-1", "a@b.com", "홍길동")
	if !errors.Is(err, domain.ErrCustomerExists) {
		t.Fatalf("중복 등록은 ErrCustomerExists 여야 하는데: %v", err)
	}
}

func TestRegister_이메일이_없으면_거부된다(t *testing.T) {
	svc := NewCustomerService(newFakeRepo(), &fakePublisher{})
	if err := svc.Register(context.Background(), "cust-1", "", "홍길동"); !errors.Is(err, domain.ErrInvalidCustomer) {
		t.Fatalf("빈 이메일은 ErrInvalidCustomer 여야 하는데: %v", err)
	}
}
