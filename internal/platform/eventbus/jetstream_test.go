package eventbus_test

import (
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// JetStream 의 핵심: 구독자가 붙기 "전에" 발행해도, 나중에 구독하면 받는다.
// (core NATS 라면 이 메시지는 그냥 사라진다 — at-most-once.)
func TestJetStream_구독전에_발행해도_나중에_받는다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	bus, err := eventbus.Connect(url, eventbus.WithJetStream())
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	// 아직 아무도 구독하지 않은 상태에서 먼저 발행한다(스트림에 저장된다).
	env, _ := eventbus.NewEnvelope("order.placed", map[string]string{"order_id": "order-1"})
	if err := bus.Publish("ordering.order.placed", env); err != nil {
		t.Fatalf("발행: %v", err)
	}

	// 발행 뒤에 구독을 건다.
	got := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.placed", "test", func(e eventbus.Envelope) error {
		got <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-got:
		var p struct {
			OrderID string `json:"order_id"`
		}
		if err := e.Into(&p); err != nil {
			t.Fatal(err)
		}
		if p.OrderID != "order-1" {
			t.Fatalf("payload 불일치: %+v", p)
		}
		// 봉투 ID 가 비어 있어도, JetStream 시퀀스로 멱등 키가 채워져야 한다.
		if e.ID == "" {
			t.Fatal("JetStream 소비 시 이벤트 ID(시퀀스)가 채워져야 함")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("타임아웃: 구독 전에 발행한 이벤트를 받지 못함(JetStream 영속 실패)")
	}
}
