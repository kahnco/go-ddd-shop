// Package ratelimit 는 키(사용자·IP)별 토큰 버킷 레이트 리미터다.
// 이벤트·상금처럼 사람이 몰리는 입구에서, 한 주체가 요청을 쏟아붓는 것을 막는다.
package ratelimit

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// Limiter 는 키별 토큰 버킷이다. 버킷은 용량 burst 로 시작해 초당 refill 개씩 채워지고,
// 요청마다 토큰 1개를 소비한다. 토큰이 없으면 거절한다.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	burst   float64
	refill  float64 // 초당 채워지는 토큰 수
	now     func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// New 는 버스트 burst, 초당 perSecond 개로 회복하는 리미터를 만든다.
func New(burst int, perSecond float64) *Limiter {
	return &Limiter{
		buckets: map[string]*bucket{},
		burst:   float64(burst),
		refill:  perSecond,
		now:     time.Now,
	}
}

// WithClock 은 테스트에서 현재 시각을 주입한다.
func (l *Limiter) WithClock(now func() time.Time) *Limiter {
	l.now = now
	return l
}

// Allow 는 key 에 토큰이 있으면 하나 쓰고 true, 없으면 false 를 반환한다.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}
	// 지난 시간만큼 토큰을 보충(버킷 용량 상한).
	if elapsed := now.Sub(b.last).Seconds(); elapsed > 0 {
		b.tokens += elapsed * l.refill
		if b.tokens > l.burst {
			b.tokens = l.burst
		}
		b.last = now
	}
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Middleware 는 keyFn 으로 키를 뽑아, 한도를 넘으면 429 를 돌려주는 HTTP 미들웨어다.
// keyFn 이 빈 문자열을 주면(식별 불가) 통과시킨다.
func (l *Limiter) Middleware(keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if key := keyFn(r); key != "" && !l.Allow(key) {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate_limited"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP 는 요청의 클라이언트 IP 를 뽑는다(프록시 뒤라면 X-Forwarded-For 우선).
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := indexComma(xff); i >= 0 {
			return trimSpace(xff[:i])
		}
		return trimSpace(xff)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func indexComma(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}
