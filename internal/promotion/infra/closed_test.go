package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/distlock"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

func TestMarkClosed_당첨후_한번만_종료하고_이후응모는_거부(t *testing.T) {
	ctx := context.Background()
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: 1, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)

	// 당첨 전엔 종료되지 않는다
	if closed, _ := svc.MarkClosed(ctx, "ev"); closed {
		t.Fatalf("당첨 전엔 종료되면 안 됨")
	}

	// 1번째 = 당첨(target=1)
	if r, _ := svc.Enter(ctx, "ev", "alice"); !r.Winner {
		t.Fatalf("alice 당첨이어야")
	}

	// 종료 — 처음이면 true
	if closed, err := svc.MarkClosed(ctx, "ev"); err != nil || !closed {
		t.Fatalf("당첨 후 종료는 true 여야: closed=%v err=%v", closed, err)
	}
	// 멱등 — 두 번째는 false
	if closed, _ := svc.MarkClosed(ctx, "ev"); closed {
		t.Fatalf("이미 종료된 이벤트는 다시 종료되면 안 됨")
	}
	// 종료 후 응모는 거부
	if _, err := svc.Enter(ctx, "ev", "bob"); err != domain.ErrClosed {
		t.Fatalf("종료 후 응모는 ErrClosed 여야: %v", err)
	}
	// 아웃박스에 EventClosed 가 적재됐는지(승자 이벤트 + 종료 이벤트 = 2건)
	n, _ := repo.DispatchOutbox(ctx, func(infra.OutboxMessage) error { return nil })
	if n != 2 {
		t.Fatalf("아웃박스에 winner+closed 2건이어야: %d", n)
	}
}

func TestCloser_당첨후_주기적으로_종료배치를_실행한다(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: 1, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)
	_, _ = svc.Enter(ctx, "ev", "alice") // 당첨 확정

	// Nop 잠금으로 Closer 를 구동 — 주기적으로 종료를 시도해 이벤트를 종료한다.
	go infra.NewCloser(svc, distlock.Nop{}, 15*time.Millisecond, discardLogger()).Watch("ev").Run(ctx)

	// 종료되면 이후 응모가 ErrClosed 로 막힌다 — 그걸로 종료 여부를 확인한다.
	deadline := time.After(2 * time.Second)
	for {
		if _, err := repo.Enter(ctx, "ev", "probe", time.Unix(1, 0)); err == domain.ErrClosed {
			return // Closer 가 종료시킴 → 이후 응모가 막힘
		}
		select {
		case <-deadline:
			t.Fatal("Closer 가 종료 배치를 실행하지 않음")
		default:
			time.Sleep(15 * time.Millisecond)
		}
	}
}
