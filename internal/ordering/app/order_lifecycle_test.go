package app

import (
	"context"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

func TestConfirmPaidOrder_결제완료면_확정되고_이벤트발행(t *testing.T) {
	svc, repo, pub := newService()
	id, err := svc.PlaceOrder(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	if err := svc.ConfirmPaidOrder(context.Background(), id); err != nil {
		t.Fatalf("ConfirmPaidOrder: %v", err)
	}

	got, _ := repo.FindByID(context.Background(), id)
	if got.Status() != domain.StatusConfirmed {
		t.Fatalf("상태 = CONFIRMED 여야 하는데 %s", got.Status())
	}
	var paid, confirmed bool
	for _, e := range pub.published {
		switch e.(type) {
		case domain.OrderPaid:
			paid = true
		case domain.OrderConfirmed:
			confirmed = true
		}
	}
	if !paid || !confirmed {
		t.Fatalf("OrderPaid·OrderConfirmed 가 발행돼야: %+v", pub.published)
	}
}

func TestCancelOrder_취소되고_이벤트발행(t *testing.T) {
	svc, repo, pub := newService()
	id, _ := svc.PlaceOrder(context.Background(), validCommand())

	if err := svc.CancelOrder(context.Background(), id); err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}

	got, _ := repo.FindByID(context.Background(), id)
	if got.Status() != domain.StatusCancelled {
		t.Fatalf("상태 = CANCELLED 여야 하는데 %s", got.Status())
	}
	var cancelled bool
	for _, e := range pub.published {
		if _, ok := e.(domain.OrderCancelled); ok {
			cancelled = true
		}
	}
	if !cancelled {
		t.Fatalf("OrderCancelled 가 발행돼야: %+v", pub.published)
	}
}
