package ratelimit

import (
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// tokenBucketScript 는 Redis 에서 토큰 버킷을 원자적으로 계산하는 Lua 스크립트다.
// 여러 replica 가 같은 Redis 를 보므로, 레이트 리밋이 인스턴스 경계를 넘어 공유된다.
// KEYS[1]=버킷 키, ARGV=[burst, refillPerSec, now(ms), requested]
const tokenBucketScript = `
local burst  = tonumber(ARGV[1])
local refill = tonumber(ARGV[2])
local now    = tonumber(ARGV[3])
local want   = tonumber(ARGV[4])

local data   = redis.call('HMGET', KEYS[1], 'tokens', 'ts')
local tokens = tonumber(data[1])
local ts     = tonumber(data[2])
if tokens == nil then
  tokens = burst
  ts = now
end

local delta = math.max(0, now - ts) / 1000.0
tokens = math.min(burst, tokens + delta * refill)

local allowed = 0
if tokens >= want then
  tokens = tokens - want
  allowed = 1
end

redis.call('HSET', KEYS[1], 'tokens', tokens, 'ts', now)
-- 놀고 있는 키가 스스로 사라지게 TTL 을 넉넉히 준다(버킷이 가득 차는 시간 + 여유).
local ttl = math.ceil(burst / refill * 1000) + 1000
redis.call('PEXPIRE', KEYS[1], ttl)
return allowed
`

// RedisLimiter 는 Redis 백엔드 토큰 버킷 리미터다(여러 replica 공유).
// 인메모리 Limiter 와 같은 의미(버스트 burst, 초당 refill)지만, 상태가 Redis 에 있다.
type RedisLimiter struct {
	client redis.Scripter
	script *redis.Script
	burst  int
	refill float64
	prefix string
	now    func() time.Time
}

func NewRedis(client redis.Scripter, burst int, perSecond float64) *RedisLimiter {
	return &RedisLimiter{
		client: client,
		script: redis.NewScript(tokenBucketScript),
		burst:  burst,
		refill: perSecond,
		prefix: "ratelimit:",
		now:    time.Now,
	}
}

// WithClock 은 테스트에서 현재 시각을 주입한다.
func (l *RedisLimiter) WithClock(now func() time.Time) *RedisLimiter {
	l.now = now
	return l
}

// Allow 는 key 에 토큰이 있으면 하나 쓰고 true. Redis 오류는 그대로 반환한다.
func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	nowMs := l.now().UnixMilli()
	res, err := l.script.Run(ctx, l.client, []string{l.prefix + key}, l.burst, l.refill, nowMs, 1).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// Middleware 는 keyFn 으로 키를 뽑아 초과 시 429 를 돌려준다.
// Redis 자체가 오류를 내면 통과시킨다(fail-open) — 리미터 장애가 서비스 전체를 막지 않게.
func (l *RedisLimiter) Middleware(keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key != "" {
				allowed, err := l.Allow(r.Context(), key)
				if err == nil && !allowed {
					w.Header().Set("Retry-After", "1")
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"error":"rate_limited"}`))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
