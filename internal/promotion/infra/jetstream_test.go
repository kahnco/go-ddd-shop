package infra_test

import (
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// JetStream 내구성: promotion.entry_requested 를 스트림이 잡으므로, 구독 "이전"에 발행해도
// 스트림에 남아 있다가 나중에 붙은 내구 소비자에게 전달된다(접수 유실 방지).
func TestJetStream_접수는_구독전_발행도_잃지_않는다(t *testing.T) {
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

	// 구독자가 아직 없는 상태에서 접수 이벤트를 발행(= 스트림에 저장).
	env, _ := eventbus.NewEnvelope("entry_requested", map[string]string{"event_id": "ev", "user_id": "alice"})
	env.ID = "req-1"
	if err := bus.Publish("promotion.entry_requested", env); err != nil {
		t.Fatalf("발행 실패(스트림 없음?): %v", err)
	}

	// 이제 내구 소비자를 붙인다 — 스트림에 남아 있던 메시지를 받아야 한다.
	got := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("promotion.entry_requested", "promotion", func(e eventbus.Envelope) error {
		got <- e
		return nil
	}); err != nil {
		t.Fatalf("구독 실패: %v", err)
	}

	select {
	case e := <-got:
		if e.ID != "req-1" {
			t.Fatalf("봉투 ID 이상: %s", e.ID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("구독 전 발행한 접수가 전달되지 않음(스트림 미persist)")
	}
}
