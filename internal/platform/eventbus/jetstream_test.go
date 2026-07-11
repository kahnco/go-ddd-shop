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

// 같은 봉투 ID(=Nats-Msg-Id)로 두 번 발행하면, JetStream 이 발행측에서 중복을 제거한다.
func TestJetStream_같은_ID로_두번_발행하면_한번만_전달된다(t *testing.T) {
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

	got := make(chan eventbus.Envelope, 4)
	if err := bus.Subscribe("ordering.order.placed", "test", func(e eventbus.Envelope) error {
		got <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	env, _ := eventbus.NewEnvelope("order.placed", map[string]string{"order_id": "order-1"})
	env.ID = "evt-1"
	if err := bus.Publish("ordering.order.placed", env); err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish("ordering.order.placed", env); err != nil { // 같은 ID → 중복 제거
		t.Fatal(err)
	}

	// 첫 번째는 받는다.
	select {
	case <-got:
	case <-time.After(3 * time.Second):
		t.Fatal("타임아웃: 첫 메시지를 받지 못함")
	}
	// 두 번째(중복)는 오지 않아야 한다.
	select {
	case <-got:
		t.Fatal("같은 Nats-Msg-Id 의 중복이 전달됨(발행측 dedup 실패)")
	case <-time.After(1 * time.Second):
		// 중복 없음 — 정상
	}
}
