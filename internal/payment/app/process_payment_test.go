package app

import (
	"context"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/payment/domain"
)

type fakeRepo struct{ saved []*domain.Payment }

func (r *fakeRepo) Save(_ context.Context, p *domain.Payment) error {
	r.saved = append(r.saved, p)
	return nil
}

type fakePublisher struct{ published []domain.DomainEvent }

func (p *fakePublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	p.published = append(p.published, events...)
	return nil
}

func TestOnStockReserved_결제되고_완료이벤트발행(t *testing.T) {
	repo := &fakeRepo{}
	pub := &fakePublisher{}
	svc := NewPaymentService(repo, pub)

	err := svc.OnStockReserved(context.Background(), ProcessPaymentCommand{OrderID: "order-1", Amount: 5000})
	if err != nil {
		t.Fatalf("OnStockReserved: %v", err)
	}
	if len(repo.saved) != 1 {
		t.Fatalf("결제 1건 저장돼야 하는데 %d건", len(repo.saved))
	}
	if _, ok := pub.published[0].(domain.PaymentCompleted); !ok {
		t.Fatalf("PaymentCompleted 발행돼야 함: %T", pub.published[0])
	}
}
