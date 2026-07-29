package infra

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// MemoryRepo 는 응모를 인메모리로 기록하는 어댑터다.
// 카운터·응모기록·당첨자·아웃박스를 하나의 뮤텍스 아래 두어, 순번 배정과 당첨 이벤트 적재를
// 원자적으로(빈틈없이) 직렬화한다. 단일 프로세스 안에서만 안전하다 — 확장하려면 PostgresRepo.
type MemoryRepo struct {
	mu      sync.Mutex
	events  map[string]domain.Event
	count   map[string]int    // eventID -> 지금까지 배정된 최대 순번(빈틈 없음)
	entries map[string]int    // "eventID|userID" -> 배정된 순번
	winner  map[string]string // eventID -> 당첨 userID
	outbox  []OutboxMessage
	seen    map[string]struct{} // 아웃박스 dedup_id 중복 방지
	nextID  int64
}

func NewMemoryRepo() *MemoryRepo {
	return &MemoryRepo{
		events:  map[string]domain.Event{},
		count:   map[string]int{},
		entries: map[string]int{},
		winner:  map[string]string{},
		seen:    map[string]struct{}{},
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
		r.enqueueWinner(eventID, userID, next, now)
	}
	return app.Result{Seq: next, Winner: winner, Already: false}, nil
}

// enqueueWinner 는 당첨 이벤트를 아웃박스에 적재한다(뮤텍스 아래 = Enter 와 같은 임계구역).
func (r *MemoryRepo) enqueueWinner(eventID, userID string, seq int, now time.Time) {
	evt := domain.WinnerDetermined{EventID: eventID, UserID: userID, Seq: seq, DeterminedAt: now}
	if _, dup := r.seen[evt.DedupID()]; dup {
		return
	}
	payload, _ := json.Marshal(evt)
	r.nextID++
	r.outbox = append(r.outbox, OutboxMessage{
		ID:        r.nextID,
		Subject:   "promotion." + evt.EventName(),
		EventName: evt.EventName(),
		Payload:   payload,
		DedupID:   evt.DedupID(),
	})
	r.seen[evt.DedupID()] = struct{}{}
}

func (r *MemoryRepo) EntryOf(_ context.Context, eventID, userID string) (app.Result, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ev, ok := r.events[eventID]
	if !ok {
		return app.Result{}, false, domain.ErrEventNotFound
	}
	seq, done := r.entries[entryKey(eventID, userID)]
	if !done {
		return app.Result{}, false, nil
	}
	return app.Result{Seq: seq, Winner: ev.IsWinner(seq), Already: true}, true, nil
}

// DispatchOutbox 는 미발행 아웃박스 메시지를 순서대로 publish 하고, 성공한 것만 발행 표시한다.
func (r *MemoryRepo) DispatchOutbox(_ context.Context, publish func(OutboxMessage) error) (int, error) {
	r.mu.Lock()
	pending := append([]OutboxMessage(nil), r.outbox...)
	r.mu.Unlock()

	sent := 0
	for _, m := range pending {
		if err := publish(m); err != nil {
			break // 실패하면 멈추고 다음 주기에 재시도
		}
		sent++
	}
	if sent > 0 {
		r.mu.Lock()
		r.outbox = r.outbox[sent:]
		r.mu.Unlock()
	}
	return sent, nil
}

// WinnerOf 는 확정된 당첨자를 돌려준다(테스트·조회용).
func (r *MemoryRepo) WinnerOf(eventID string) (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.winner[eventID]
	return w, ok
}

var (
	_ app.Repository = (*MemoryRepo)(nil)
	_ OutboxStore    = (*MemoryRepo)(nil)
)
