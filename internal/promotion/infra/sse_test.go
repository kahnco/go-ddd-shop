package infra_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

func TestSSE_소비자가_처리하면_구독자에게_실시간_push(t *testing.T) {
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(context.Background(), domain.Event{ID: "ev", TargetSeq: 1, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)
	hub := infra.NewHub()
	consumer := infra.NewEntryRequestedConsumer(svc, discardLogger()).WithHub(hub)

	ch := hub.Subscribe("ev|alice") // Hub 키 = eventID|userID
	defer hub.Unsubscribe("ev|alice", ch)

	if err := consumer.Handle(requestEnvelope("r1", "ev", "alice")); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	select {
	case data := <-ch:
		var body struct {
			Seq    int  `json:"seq"`
			Winner bool `json:"winner"`
		}
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatalf("push 페이로드 디코딩: %v", err)
		}
		if body.Seq != 1 || !body.Winner { // target=1 이라 1번째가 당첨
			t.Fatalf("push 내용 이상: seq=%d winner=%v", body.Seq, body.Winner)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("처리 결과가 구독자에게 push 되지 않음")
	}
}

func TestSSE_사용자없으면_401(t *testing.T) {
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(context.Background(), domain.Event{ID: "ev", TargetSeq: 1000, StartsAt: time.Unix(0, 0)})
	mux := http.NewServeMux()
	infra.NewSSEHandler(app.NewService(repo), infra.NewHub()).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/events/ev/entries/me/stream", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("상태 %d (401 기대)", rec.Code)
	}
}
