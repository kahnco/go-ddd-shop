package domain

import (
	"errors"
	"testing"
)

// 테스트 헬퍼: 유효한 값 객체를 간결하게 만든다.
func mustMoney(t *testing.T, amount int64) Money {
	t.Helper()
	m, err := NewMoney(amount)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", amount, err)
	}
	return m
}

func mustQty(t *testing.T, v int) Quantity {
	t.Helper()
	q, err := NewQuantity(v)
	if err != nil {
		t.Fatalf("NewQuantity(%d): %v", v, err)
	}
	return q
}

// 상품 A 2개(1,000원) + 상품 B 1개(3,000원) = 5,000원짜리 주문 라인.
func sampleLines(t *testing.T) []OrderLine {
	t.Helper()
	return []OrderLine{
		NewOrderLine("prod-A", mustQty(t, 2), mustMoney(t, 1000)),
		NewOrderLine("prod-B", mustQty(t, 1), mustMoney(t, 3000)),
	}
}

func TestPlaceOrder_빈_항목이면_에러(t *testing.T) {
	_, err := PlaceOrder("order-1", "cust-1", nil)
	if !errors.Is(err, ErrEmptyOrder) {
		t.Fatalf("빈 주문은 ErrEmptyOrder 여야 하는데: %v", err)
	}
}

func TestPlaceOrder_총액은_소계의_합(t *testing.T) {
	o, err := PlaceOrder("order-1", "cust-1", sampleLines(t))
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if got := o.Total().Amount(); got != 5000 {
		t.Fatalf("총액 = 5000 이어야 하는데 %d", got)
	}
}

func TestPlaceOrder_OrderPlaced_이벤트를_기록(t *testing.T) {
	o, _ := PlaceOrder("order-1", "cust-1", sampleLines(t))
	events := o.PullEvents()
	if len(events) != 1 {
		t.Fatalf("이벤트 1개 여야 하는데 %d개", len(events))
	}
	placed, ok := events[0].(OrderPlaced)
	if !ok {
		t.Fatalf("첫 이벤트는 OrderPlaced 여야 함: %T", events[0])
	}
	if placed.OrderID != "order-1" || placed.Total.Amount() != 5000 {
		t.Fatalf("이벤트 내용 불일치: %+v", placed)
	}
	// PullEvents 후에는 비어야 한다.
	if again := o.PullEvents(); len(again) != 0 {
		t.Fatalf("PullEvents 후에는 비어야 하는데 %d개", len(again))
	}
}

func TestOrder_정상_상태_전이(t *testing.T) {
	o, _ := PlaceOrder("order-1", "cust-1", sampleLines(t))
	steps := []struct {
		name string
		do   func() error
		want OrderStatus
	}{
		{"결제", o.MarkPaid, StatusPaid},
		{"확정", o.Confirm, StatusConfirmed},
		{"배송", o.Ship, StatusShipped},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("%s 전이 실패: %v", s.name, err)
		}
		if o.Status() != s.want {
			t.Fatalf("%s 후 상태 = %s 여야 하는데 %s", s.name, s.want, o.Status())
		}
	}
}

func TestOrder_허용되지_않은_전이는_거부(t *testing.T) {
	// 생성됨 → (결제 없이) 확정 : 거부돼야 함
	o, _ := PlaceOrder("order-1", "cust-1", sampleLines(t))
	if err := o.Confirm(); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("PLACED→CONFIRMED 는 거부돼야 하는데: %v", err)
	}
	if o.Status() != StatusPlaced {
		t.Fatalf("거부됐으면 상태는 PLACED 그대로여야 하는데 %s", o.Status())
	}

	// 확정된 주문은 취소할 수 없다(PLACED/PAID 에서만 취소 가능)
	_ = o.MarkPaid()
	_ = o.Confirm()
	if err := o.Cancel(); !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("CONFIRMED→CANCELLED 는 거부돼야 하는데: %v", err)
	}
}

func TestOrder_생성_직후_취소_가능(t *testing.T) {
	o, _ := PlaceOrder("order-1", "cust-1", sampleLines(t))
	if err := o.Cancel(); err != nil {
		t.Fatalf("PLACED 에서 취소는 가능해야 하는데: %v", err)
	}
	if o.Status() != StatusCancelled {
		t.Fatalf("취소 후 상태 = CANCELLED 여야 하는데 %s", o.Status())
	}
}

func TestMoney_음수는_에러(t *testing.T) {
	if _, err := NewMoney(-1); !errors.Is(err, ErrNegativeMoney) {
		t.Fatalf("음수 금액은 ErrNegativeMoney 여야 하는데: %v", err)
	}
}

func TestQuantity_0이하는_에러(t *testing.T) {
	for _, v := range []int{0, -3} {
		if _, err := NewQuantity(v); !errors.Is(err, ErrNonPositiveQuantity) {
			t.Fatalf("수량 %d 는 ErrNonPositiveQuantity 여야 하는데: %v", v, err)
		}
	}
}
