package ratelimit_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/ratelimit"
)

func TestLimiter_버스트까지_허용하고_초과는_거절(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(3, 1).WithClock(func() time.Time { return now })

	for i := 0; i < 3; i++ {
		if !l.Allow("u1") {
			t.Fatalf("버스트 안(%d번째)은 허용돼야", i+1)
		}
	}
	if l.Allow("u1") {
		t.Fatalf("버스트 초과는 거절돼야")
	}
}

func TestLimiter_시간이_지나면_회복된다(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(1, 1).WithClock(func() time.Time { return now })

	if !l.Allow("u1") {
		t.Fatal("첫 요청 허용")
	}
	if l.Allow("u1") {
		t.Fatal("두 번째는 즉시 거절")
	}
	now = now.Add(time.Second) // 1초 뒤 토큰 1개 회복
	if !l.Allow("u1") {
		t.Fatal("1초 뒤엔 회복돼 허용")
	}
}

func TestLimiter_키마다_독립적이다(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(1, 1).WithClock(func() time.Time { return now })
	if !l.Allow("a") || !l.Allow("b") {
		t.Fatal("서로 다른 키는 각자 버킷")
	}
	if l.Allow("a") {
		t.Fatal("a 는 소진")
	}
}

func TestMiddleware_초과하면_429(t *testing.T) {
	now := time.Unix(0, 0)
	l := ratelimit.New(1, 1).WithClock(func() time.Time { return now })
	key := func(r *http.Request) string { return r.Header.Get("X-User-Id") }
	h := l.Middleware(key)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("X-User-Id", "u1")

	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("첫 요청 %d (200 기대)", rec1.Code)
	}
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("두 번째 %d (429 기대)", rec2.Code)
	}
}
