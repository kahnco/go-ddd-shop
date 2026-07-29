package app_test

import (
	"context"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

func newSvc(t *testing.T, target int) (*app.Service, *infra.MemoryRepo) {
	t.Helper()
	repo := infra.NewMemoryRepo()
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: "ev", TargetSeq: target, StartsAt: time.Unix(0, 0),
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	return app.NewService(repo), repo
}

// 아웃박스에 쌓인 당첨 이벤트 수를 센다.
func drained(t *testing.T, repo *infra.MemoryRepo) int {
	t.Helper()
	n, err := repo.DispatchOutbox(context.Background(), func(infra.OutboxMessage) error { return nil })
	if err != nil {
		t.Fatalf("DispatchOutbox: %v", err)
	}
	return n
}

func TestEnter_당첨이_확정되면_아웃박스에_한번_적재된다(t *testing.T) {
	svc, repo := newSvc(t, 2)

	// 1번째 — 당첨 아님, 아웃박스 비어 있음
	if r, err := svc.Enter(context.Background(), "ev", "alice"); err != nil || r.Winner {
		t.Fatalf("alice: seq=%d winner=%v err=%v", r.Seq, r.Winner, err)
	}
	// 2번째 — 당첨(target=2) → 아웃박스에 1건 적재
	r, err := svc.Enter(context.Background(), "ev", "bob")
	if err != nil {
		t.Fatalf("bob: %v", err)
	}
	if !r.Winner || r.Seq != 2 {
		t.Fatalf("bob 는 당첨(seq=2)이어야: seq=%d winner=%v", r.Seq, r.Winner)
	}

	// 당첨자가 다시 응모(멱등) — 아웃박스에 더 쌓이지 않는다
	r2, err := svc.Enter(context.Background(), "ev", "bob")
	if err != nil {
		t.Fatalf("bob 재응모: %v", err)
	}
	if !r2.Already || r2.Seq != 2 {
		t.Fatalf("bob 재응모는 Already·seq=2 여야: %+v", r2)
	}

	if n := drained(t, repo); n != 1 {
		t.Fatalf("당첨 이벤트는 아웃박스에 정확히 1건이어야: %d", n)
	}
}

func TestEnter_시작전이면_거부하고_순번을_소비하지않는다(t *testing.T) {
	repo := infra.NewMemoryRepo()
	future := time.Now().Add(time.Hour)
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: "ev", TargetSeq: 1000, StartsAt: future,
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	svc := app.NewService(repo).WithClock(func() time.Time { return future.Add(-time.Minute) })

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

func TestEntryOf_상태를_조회한다(t *testing.T) {
	svc, _ := newSvc(t, 1000)

	if _, found, _ := svc.EntryOf(context.Background(), "ev", "alice"); found {
		t.Fatalf("응모 전엔 found=false 여야")
	}
	if _, err := svc.Enter(context.Background(), "ev", "alice"); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	res, found, err := svc.EntryOf(context.Background(), "ev", "alice")
	if err != nil || !found || res.Seq != 1 {
		t.Fatalf("응모 후엔 found=true·seq=1 여야: res=%+v found=%v err=%v", res, found, err)
	}
}
