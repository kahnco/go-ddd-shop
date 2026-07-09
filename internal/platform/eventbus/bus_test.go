package eventbus_test

import (
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// 임베디드 NATS 위에서 발행 → 구독 왕복이 봉투를 온전히 전달하는지 본다.
func TestBus_발행하면_구독자가_봉투를_받는다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	bus, err := eventbus.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	type payload struct {
		Hello string `json:"hello"`
	}
	got := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("demo.hi", "test", func(env eventbus.Envelope) error {
		got <- env
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	env, err := eventbus.NewEnvelope("demo.hi", payload{Hello: "world"})
	if err != nil {
		t.Fatal(err)
	}
	if err := bus.Publish("demo.hi", env); err != nil {
		t.Fatal(err)
	}

	select {
	case e := <-got:
		if e.Name != "demo.hi" {
			t.Fatalf("이벤트 이름 = demo.hi 여야 하는데 %s", e.Name)
		}
		var p payload
		if err := e.Into(&p); err != nil {
			t.Fatal(err)
		}
		if p.Hello != "world" {
			t.Fatalf("payload = world 여야 하는데 %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("타임아웃: 메시지를 받지 못함")
	}
}
