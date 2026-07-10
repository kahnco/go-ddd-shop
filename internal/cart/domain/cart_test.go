package domain

import (
	"errors"
	"testing"
)

func TestCart_같은상품은_수량이_합쳐진다(t *testing.T) {
	c := NewCart("cust-1")
	_ = c.AddItem("prod-A", 2)
	_ = c.AddItem("prod-A", 3)

	items := c.Items()
	if len(items) != 1 || items[0].Quantity != 5 {
		t.Fatalf("prod-A 는 5개로 합쳐져야 하는데 %+v", items)
	}
}

func TestCart_0이하_수량은_거부된다(t *testing.T) {
	c := NewCart("cust-1")
	if err := c.AddItem("prod-A", 0); !errors.Is(err, ErrNonPositiveQuantity) {
		t.Fatalf("수량 0 은 ErrNonPositiveQuantity 여야 하는데: %v", err)
	}
}

func TestCart_빼면_비워지고_IsEmpty(t *testing.T) {
	c := NewCart("cust-1")
	_ = c.AddItem("prod-A", 1)
	c.RemoveItem("prod-A")
	if !c.IsEmpty() {
		t.Fatal("빼고 나면 비어 있어야 한다")
	}
}

func TestCart_항목은_상품ID순으로_정렬된다(t *testing.T) {
	c := NewCart("cust-1")
	_ = c.AddItem("prod-B", 1)
	_ = c.AddItem("prod-A", 1)
	items := c.Items()
	if items[0].ProductID != "prod-A" || items[1].ProductID != "prod-B" {
		t.Fatalf("상품 ID 순 정렬돼야 하는데 %+v", items)
	}
}
