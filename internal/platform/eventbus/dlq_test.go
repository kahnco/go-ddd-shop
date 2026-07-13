package eventbus_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// 짧은 백오프로 재시도·DLQ 를 빠르게 검증하기 위한 헬퍼.
func newRetryBus(t *testing.T, maxDeliver int) *eventbus.Bus {
	t.Helper()
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(shutdown)
	bus, err := eventbus.Connect(url,
		eventbus.WithJetStream(),
		eventbus.WithRetry(maxDeliver, 10*time.Millisecond, 20*time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(bus.Close)
	return bus
}

func TestSubscribe_계속_실패하면_maxDeliver_뒤_DLQ로(t *testing.T) {
	bus := newRetryBus(t, 3)

	var calls atomic.Int32
	// 항상 실패하는 독성 핸들러.
	if err := bus.Subscribe("payment.completed", "ordering", func(eventbus.Envelope) error {
		calls.Add(1)
		return fmt.Errorf("일부러 실패")
	}); err != nil {
		t.Fatal(err)
	}

	// DLQ 로 떨어진 걸 잡는다.
	dead := make(chan eventbus.DeadLetter, 1)
	if err := bus.SubscribeDLQ("dlqmonitor", func(dl eventbus.DeadLetter) error {
		dead <- dl
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	env, _ := eventbus.NewEnvelope("payment.completed", map[string]string{"order_id": "order-1"})
	env.ID = "evt-1"
	if err := bus.Publish("payment.completed", env); err != nil {
		t.Fatal(err)
	}

	select {
	case dl := <-dead:
		if dl.Attempts != 3 {
			t.Errorf("시도 횟수 = 3 여야 하는데 %d", dl.Attempts)
		}
		if dl.Subject != "payment.completed" {
			t.Errorf("원래 subject 보존돼야: %s", dl.Subject)
		}
		if dl.Error == "" {
			t.Error("실패 원인이 기록돼야 함")
		}
		var p struct {
			OrderID string `json:"order_id"`
		}
		_ = dl.Event.Into(&p)
		if p.OrderID != "order-1" {
			t.Errorf("원본 봉투 보존돼야: %+v", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: DLQ 로 오지 않음")
	}

	// 정확히 maxDeliver(3)회만 시도했어야 한다(무한 재전송이 아니라).
	time.Sleep(200 * time.Millisecond)
	if got := calls.Load(); got != 3 {
		t.Fatalf("핸들러 호출 = 3회여야 하는데 %d회", got)
	}
}

func TestSubscribe_일시적_실패는_재시도로_성공한다(t *testing.T) {
	bus := newRetryBus(t, 5)

	var calls atomic.Int32
	// 처음 두 번은 실패, 세 번째에 성공(일시적 장애 흉내).
	done := make(chan struct{}, 1)
	if err := bus.Subscribe("payment.completed", "ordering", func(eventbus.Envelope) error {
		if calls.Add(1) < 3 {
			return fmt.Errorf("일시적 장애")
		}
		done <- struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// DLQ 에 오면 안 된다(성공했으니).
	dlqHit := make(chan struct{}, 1)
	if err := bus.SubscribeDLQ("dlqmonitor", func(eventbus.DeadLetter) error {
		dlqHit <- struct{}{}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	env, _ := eventbus.NewEnvelope("payment.completed", map[string]string{"order_id": "order-2"})
	env.ID = "evt-2"
	if err := bus.Publish("payment.completed", env); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
		// 성공. DLQ 에 안 왔는지 잠깐 확인.
		select {
		case <-dlqHit:
			t.Fatal("성공한 메시지가 DLQ 로 오면 안 됨")
		case <-time.After(300 * time.Millisecond):
		}
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: 재시도로 성공하지 못함")
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("핸들러 호출 = 3회여야 하는데 %d회", got)
	}
}
