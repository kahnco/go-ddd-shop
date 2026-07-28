package app_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// 발행된 이벤트를 담아 두는 가짜 퍼블리셔.
type capturePublisher struct {
	mu     sync.Mutex
	events []domain.DomainEvent
}

func (c *capturePublisher) Publish(_ context.Context, events ...domain.DomainEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, events...)
	return nil
}

func (c *capturePublisher) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

func newSvc(t *testing.T, target int) (*app.Service, *capturePublisher) {
	t.Helper()
	repo := infra.NewMemoryRepo()
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: "ev", TargetSeq: target, StartsAt: time.Unix(0, 0),
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	pub := &capturePublisher{}
	return app.NewService(repo, pub), pub
}

func TestEnter_당첨자에게만_이벤트를_한번_발행한다(t *testing.T) {
	svc, pub := newSvc(t, 2)

	// 1번째 — 당첨 아님
	if r, err := svc.Enter(context.Background(), "ev", "alice"); err != nil || r.Winner {
		t.Fatalf("alice: seq=%d winner=%v err=%v", r.Seq, r.Winner, err)
	}
	// 2번째 — 당첨(target=2)
	r, err := svc.Enter(context.Background(), "ev", "bob")
	if err != nil {
		t.Fatalf("bob: %v", err)
	}
	if !r.Winner || r.Seq != 2 {
		t.Fatalf("bob 는 당첨(seq=2)이어야: seq=%d winner=%v", r.Seq, r.Winner)
	}
	if pub.count() != 1 {
		t.Fatalf("당첨 이벤트는 1번 발행돼야: %d", pub.count())
	}

	// 당첨자가 다시 응모(멱등) — 재발행하지 않는다
	r2, err := svc.Enter(context.Background(), "ev", "bob")
	if err != nil {
		t.Fatalf("bob 재응모: %v", err)
	}
	if !r2.Already || r2.Seq != 2 {
		t.Fatalf("bob 재응모는 Already·seq=2 여야: %+v", r2)
	}
	if pub.count() != 1 {
		t.Fatalf("멱등 재응모는 재발행하지 않아야: %d", pub.count())
	}
}

func TestEnter_시작전이면_거부한다(t *testing.T) {
	repo := infra.NewMemoryRepo()
	future := time.Now().Add(time.Hour)
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: "ev", TargetSeq: 1000, StartsAt: future,
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	// 현재 시각을 시작 이전으로 고정
	svc := app.NewService(repo, nil).WithClock(func() time.Time { return future.Add(-time.Minute) })

	if _, err := svc.Enter(context.Background(), "ev", "alice"); err != domain.ErrNotStarted {
		t.Fatalf("시작 전 응모는 ErrNotStarted 여야: %v", err)
	}
	// 거부된 응모가 순번을 소비하지 않았는지 — 시작 후 첫 응모가 1번이어야
	svc = svc.WithClock(func() time.Time { return future.Add(time.Minute) })
	r, err := svc.Enter(context.Background(), "ev", "alice")
	if err != nil {
		t.Fatalf("시작 후 응모: %v", err)
	}
	if r.Seq != 1 {
		t.Fatalf("거부된 응모가 순번을 소비함 — 첫 순번이 %d (1이어야)", r.Seq)
	}
}
