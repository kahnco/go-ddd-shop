package infra_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// 다중 인스턴스 팬아웃 검증: 한 인스턴스(브로드캐스터)가 배정 결과를 뿌리면,
// 다른 인스턴스(브리지+허브)가 받아 자기 SSE 구독자에게 push 한다.
func TestFanout_브로드캐스트된_결과가_다른인스턴스_허브로_전달된다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()

	bus, err := eventbus.Connect(url) // core 모드(notify 는 영속 불필요)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	// 인스턴스 B: 허브 + 브리지(팬아웃 수신 → 로컬 허브 push)
	hub := infra.NewHub()
	ch := hub.Subscribe("ev|alice")
	defer hub.Unsubscribe("ev|alice", ch)
	if err := infra.NewNotifyBridge(bus, hub, discardLogger()).Start(); err != nil {
		t.Fatal(err)
	}

	// 인스턴스 A: 소비자가 결과를 브로드캐스트
	notifier := infra.NewBroadcastNotifier(bus)
	notifier.NotifyEntry("ev", "alice", app.Result{Seq: 1000, Winner: true})

	select {
	case data := <-ch:
		var body struct {
			Seq    int  `json:"seq"`
			Winner bool `json:"winner"`
		}
		_ = json.Unmarshal(data, &body)
		if body.Seq != 1000 || !body.Winner {
			t.Fatalf("팬아웃 전달 내용 이상: seq=%d winner=%v", body.Seq, body.Winner)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("브로드캐스트가 다른 인스턴스 허브로 전달되지 않음")
	}
}
