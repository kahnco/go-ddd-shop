package domain

import (
	"errors"
	"testing"
)

func TestStockItem_예약하면_가용재고가_줄어든다(t *testing.T) {
	item := NewStockItem("prod-A", 10)
	if err := item.Reserve(3); err != nil {
		t.Fatalf("예약 실패: %v", err)
	}
	if item.Available() != 7 {
		t.Fatalf("예약 후 가용 = 7 이어야 하는데 %d", item.Available())
	}
}

func TestStockItem_재고보다_많이_예약하면_거부된다(t *testing.T) {
	item := NewStockItem("prod-A", 2)
	if err := item.Reserve(5); !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("초과 예약은 ErrInsufficientStock 여야 하는데: %v", err)
	}
	// 거부됐으면 재고는 그대로여야 한다.
	if item.Available() != 2 {
		t.Fatalf("거부 후 가용 = 2 그대로여야 하는데 %d", item.Available())
	}
}

func TestStockItem_0이하_수량은_거부된다(t *testing.T) {
	item := NewStockItem("prod-A", 10)
	if err := item.Reserve(0); !errors.Is(err, ErrNonPositiveQuantity) {
		t.Fatalf("수량 0 은 ErrNonPositiveQuantity 여야 하는데: %v", err)
	}
}

func TestStockItem_Release_는_예약을_되돌린다(t *testing.T) {
	item := NewStockItem("prod-A", 10)
	_ = item.Reserve(4)
	item.Release(4)
	if item.Available() != 10 {
		t.Fatalf("복원 후 가용 = 10 이어야 하는데 %d", item.Available())
	}
}
