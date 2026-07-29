package infra_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func requestEnvelope(id, event, user string) eventbus.Envelope {
	env, _ := eventbus.NewEnvelope("entry_requested", struct {
		EventID   string `json:"event_id"`
		UserID    string `json:"user_id"`
		RequestID string `json:"request_id"`
	}{event, user, id})
	env.ID = id
	return env
}

func TestConsumer_요청을_처리해_순번을_배정한다_멱등(t *testing.T) {
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(context.Background(), domain.Event{ID: "ev", TargetSeq: 1000, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)
	consumer := infra.NewEntryRequestedConsumer(svc, discardLogger())

	// 같은 요청(req-1)을 두 번 처리 → 순번은 한 번만 배정
	if err := consumer.Handle(requestEnvelope("req-1", "ev", "alice")); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if err := consumer.Handle(requestEnvelope("req-1", "ev", "alice")); err != nil {
		t.Fatalf("Handle(중복): %v", err)
	}
	if res, found, _ := svc.EntryOf(context.Background(), "ev", "alice"); !found || res.Seq != 1 {
		t.Fatalf("alice 는 seq=1 이어야: %+v found=%v", res, found)
	}
	// 다음 사용자 → seq 2 (중복 처리가 순번을 새게 하지 않음)
	if err := consumer.Handle(requestEnvelope("req-2", "ev", "bob")); err != nil {
		t.Fatalf("Handle bob: %v", err)
	}
	if res, _, _ := svc.EntryOf(context.Background(), "ev", "bob"); res.Seq != 2 {
		t.Fatalf("bob 는 seq=2 이어야: %+v", res)
	}
}

// 접수 큐를 흉내내는 가짜 — 호출을 기록한다.
type fakeQueue struct {
	mu    sync.Mutex
	calls []string
}

func (q *fakeQueue) Enqueue(_ context.Context, eventID, userID, requestID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.calls = append(q.calls, eventID+"|"+userID+"|"+requestID)
	return nil
}

func TestQueuedHandler_접수하면_202와_요청ID(t *testing.T) {
	q := &fakeQueue{}
	mux := http.NewServeMux()
	infra.NewQueuedEntryHandler(q).Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/events/launch/entries", nil)
	req.Header.Set("X-User-Id", "alice")
	req.Header.Set("Idempotency-Key", "key-1")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("상태 %d (202 기대)", rec.Code)
	}
	var body struct {
		RequestID string `json:"request_id"`
		Status    string `json:"status"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.RequestID != "key-1" || body.Status != "queued" {
		t.Fatalf("응답 이상: %+v", body)
	}
	if len(q.calls) != 1 || q.calls[0] != "launch|alice|key-1" {
		t.Fatalf("큐 적재 이상: %v", q.calls)
	}
}

func TestStatusHandler_처리전엔_202_처리후엔_순번(t *testing.T) {
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(context.Background(), domain.Event{ID: "launch", TargetSeq: 1000, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)
	mux := http.NewServeMux()
	infra.NewStatusHandler(svc).Register(mux)

	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/events/launch/entries/me", nil)
		req.Header.Set("X-User-Id", "alice")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}

	if rec := get(); rec.Code != http.StatusAccepted {
		t.Fatalf("응모 전 상태 %d (202 processing 기대)", rec.Code)
	}
	if _, err := svc.Enter(context.Background(), "launch", "alice"); err != nil {
		t.Fatalf("Enter: %v", err)
	}
	rec := get()
	if rec.Code != http.StatusOK {
		t.Fatalf("응모 후 상태 %d (200 기대)", rec.Code)
	}
}
