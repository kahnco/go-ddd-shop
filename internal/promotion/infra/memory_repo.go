package infra

import (
	"context"
	"sync"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// MemoryRepo 는 응모를 인메모리로 기록하는 어댑터다.
// 카운터·응모기록·당첨자를 하나의 뮤텍스 아래 두어, 순번 배정을 원자적으로(빈틈없이)
// 직렬화한다. 단일 프로세스 안에서만 안전하다 — 여러 replica 로 확장하려면 PostgresRepo.
type MemoryRepo struct {
	mu      sync.Mutex
	events  map[string]domain.Event
	count   map[string]int    // eventID -> 지금까지 배정된 최대 순번(빈틈 없음)
	entries map[string]int    // "eventID|userID" -> 배정된 순번
	winner  map[string]string // eventID -> 당첨 userID
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		events:  map[string]domain.Event{},
		count:   map[string]int{},
		entries: map[string]int{},
		winner:  map[string]string{},
	}
}

func entryKey(eventID, userID string) string { return eventID + "|" + userID }

func (r *MemoryRepo) SeedEvent(_ context.Context, e domain.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events[e.ID] = e
	return nil
}

func (r *MemoryRepo) Enter(_ context.Context, eventID, userID string, now time.Time) (app.Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ev, ok := r.events[eventID]
	if !ok {
		return app.Result{}, domain.ErrEventNotFound
	}
	if now.Before(ev.StartsAt) {
		return app.Result{}, domain.ErrNotStarted
	}

	// 멱등: 이미 응모한 사용자면 기존 순번을 그대로(순번 미소비).
	if seq, done := r.entries[entryKey(eventID, userID)]; done {
		return app.Result{Seq: seq, Winner: ev.IsWinner(seq), Already: true}, nil
	}

	// 다음 순번 배정 — 뮤텍스 아래이므로 빈틈없이 직렬화된다.
	next := r.count[eventID] + 1
	r.count[eventID] = next
	r.entries[entryKey(eventID, userID)] = next

	winner := ev.IsWinner(next)
	if winner {
		r.winner[eventID] = userID
	}
	return app.Result{Seq: next, Winner: winner, Already: false}, nil
}

// WinnerOf 는 확정된 당첨자를 돌려준다(테스트·조회용).
func (r *MemoryRepo) WinnerOf(eventID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.winner[eventID]
	return w, ok
}

// 컴파일 타임 포트 준수 확인.
var _ app.Repository = (*MemoryRepo)(nil)
