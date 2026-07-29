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

func newHTTP(t *testing.T, target int, startsAt time.Time) http.Handler {
	t.Helper()
	repo := infra.NewMemoryRepo()
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: "launch", TargetSeq: target, StartsAt: startsAt,
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	mux := http.NewServeMux()
	infra.NewEntryHandler(app.NewService(repo)).Register(mux)
	return mux
}

func post(t *testing.T, h http.Handler, event, user string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/events/"+event+"/entries", nil)
	if user != "" {
		req.Header.Set("X-User-Id", user)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHTTP_응모하면_순번과_당첨여부를_준다(t *testing.T) {
	h := newHTTP(t, 2, time.Unix(0, 0))

	rec := post(t, h, "launch", "alice")
	if rec.Code != http.StatusOK {
		t.Fatalf("상태 %d (200 기대)", rec.Code)
	}
	var body struct {
		Seq    int  `json:"seq"`
		Winner bool `json:"winner"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Seq != 1 || body.Winner {
		t.Fatalf("alice: seq=%d winner=%v (1·false 기대)", body.Seq, body.Winner)
	}

	// 2번째(target=2) → 당첨
	rec = post(t, h, "launch", "bob")
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Seq != 2 || !body.Winner {
		t.Fatalf("bob: seq=%d winner=%v (2·true 기대)", body.Seq, body.Winner)
	}
}

func TestHTTP_사용자없으면_401(t *testing.T) {
	h := newHTTP(t, 1000, time.Unix(0, 0))
	if rec := post(t, h, "launch", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("상태 %d (401 기대)", rec.Code)
	}
}

func TestHTTP_없는이벤트면_404(t *testing.T) {
	h := newHTTP(t, 1000, time.Unix(0, 0))
	if rec := post(t, h, "no-such", "alice"); rec.Code != http.StatusNotFound {
		t.Fatalf("상태 %d (404 기대)", rec.Code)
	}
}

func TestHTTP_시작전이면_425(t *testing.T) {
	h := newHTTP(t, 1000, time.Now().Add(time.Hour))
	if rec := post(t, h, "launch", "alice"); rec.Code != http.StatusTooEarly {
		t.Fatalf("상태 %d (425 기대)", rec.Code)
	}
}
