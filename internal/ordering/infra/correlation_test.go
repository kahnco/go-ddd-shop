package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// 컨텍스트에 심은 상관 ID 가, 발행된 이벤트 봉투의 메타데이터로 실려 나가는지 본다.
// 이게 여러 서비스에 걸친 하나의 흐름을 같은 ID 로 추적할 수 있게 하는 핵심이다.
func TestNatsEventPublisher_상관ID를_이벤트에_실어보낸다(t *testing.T) {
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

	got := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.placed", "test", func(env eventbus.Envelope) error {
		got <- env
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// 요청 컨텍스트에 상관 ID 를 심는다(미들웨어가 하는 일을 흉내).
	ctx := telemetry.WithCorrelationID(context.Background(), "corr-123")
	pub := infra.NewNatsEventPublisher(bus, "ordering")
	if err := pub.Publish(ctx, domain.OrderPlaced{OrderID: "order-1"}); err != nil {
		t.Fatal(err)
	}

	select {
	case env := <-got:
		if env.Meta[telemetry.MetaCorrelationID] != "corr-123" {
			t.Fatalf("봉투 메타의 상관 ID = corr-123 여야 하는데 %q", env.Meta[telemetry.MetaCorrelationID])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("타임아웃: 이벤트를 받지 못함")
	}
}
