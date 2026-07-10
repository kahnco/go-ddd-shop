package app

import (
	"context"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
)

type fakeRepo struct {
	store map[domain.ProductID]*domain.Product
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[domain.ProductID]*domain.Product{}} }
func (r *fakeRepo) Save(_ context.Context, p *domain.Product) error {
	r.store[p.ID()] = p
	return nil
}
func (r *fakeRepo) Find(_ context.Context, id domain.ProductID) (*domain.Product, error) {
	p, ok := r.store[id]
	if !ok {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}
func (r *fakeRepo) All(_ context.Context) ([]*domain.Product, error) {
	out := make([]*domain.Product, 0, len(r.store))
	for _, p := range r.store {
		out = append(out, p)
	}
	return out, nil
}

type fakePublisher struct{ published []domain.DomainEvent }

func (p *fakePublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	p.published = append(p.published, events...)
	return nil
}

func TestAddProduct_등록되고_이벤트발행(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	svc := NewCatalogService(repo, pub)

	if err := svc.AddProduct(context.Background(), "prod-A", "사과", 1000); err != nil {
		t.Fatalf("AddProduct: %v", err)
	}
	if _, ok := pub.published[0].(domain.ProductAdded); !ok {
		t.Fatalf("ProductAdded 발행돼야 함: %T", pub.published[0])
	}
	got, _ := svc.Get(context.Background(), "prod-A")
	if got.Price() != 1000 {
		t.Fatalf("가격 = 1000 여야 하는데 %d", got.Price())
	}
}

func TestChangePrice_가격변경되고_이벤트발행(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	svc := NewCatalogService(repo, pub)
	_ = svc.AddProduct(context.Background(), "prod-A", "사과", 1000)

	if err := svc.ChangePrice(context.Background(), "prod-A", 1500); err != nil {
		t.Fatalf("ChangePrice: %v", err)
	}
	last := pub.published[len(pub.published)-1]
	changed, ok := last.(domain.ProductPriceChanged)
	if !ok {
		t.Fatalf("ProductPriceChanged 발행돼야 함: %T", last)
	}
	if changed.Price != 1500 {
		t.Fatalf("변경 가격 = 1500 여야 하는데 %d", changed.Price)
	}
}
