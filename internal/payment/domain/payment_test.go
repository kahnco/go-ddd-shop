package domain

import "testing"

func TestPayment_유효한_금액이면_완료되고_이벤트발행(t *testing.T) {
	p := NewPayment("order-1", 5000)
	p.Process()

	if p.Status() != StatusCompleted {
		t.Fatalf("상태 = COMPLETED 여야 하는데 %s", p.Status())
	}
	events := p.PullEvents()
	if len(events) != 1 {
		t.Fatalf("이벤트 1개여야 하는데 %d개", len(events))
	}
	done, ok := events[0].(PaymentCompleted)
	if !ok {
		t.Fatalf("PaymentCompleted 여야 함: %T", events[0])
	}
	if done.Amount != 5000 {
		t.Fatalf("결제 금액 = 5000 여야 하는데 %d", done.Amount)
	}
}

func TestPayment_0이하_금액이면_실패이벤트(t *testing.T) {
	p := NewPayment("order-1", 0)
	p.Process()

	if p.Status() != StatusFailed {
		t.Fatalf("상태 = FAILED 여야 하는데 %s", p.Status())
	}
	events := p.PullEvents()
	if _, ok := events[0].(PaymentFailed); !ok {
		t.Fatalf("PaymentFailed 여야 함: %T", events[0])
	}
}
