// Package distlock 은 Redis 기반 분산 잠금이다.
// 여러 replica 가 도는 상황에서 "한 번만 해야 하는 일"(예: 이벤트 종료 배치)을
// 정확히 한 인스턴스만 수행하도록 상호 배제를 제공한다.
package distlock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

// releaseScript 는 "내가 건 잠금일 때만" 푼다(토큰 비교 후 삭제). 남의 잠금을 실수로
// 풀지 않게, GET 과 DEL 을 원자적으로 묶는다.
const releaseScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then
  return redis.call('DEL', KEYS[1])
else
  return 0
end
`

// Lock 은 Redis 백엔드 분산 잠금이다. 각 잠금은 TTL 을 가져, 잠근 인스턴스가 죽어도
// 자동으로 풀린다(데드락 방지).
type Lock struct {
	client  redis.Cmdable
	ttl     time.Duration
	release *redis.Script
}

func New(client redis.Cmdable, ttl time.Duration) *Lock {
	return &Lock{client: client, ttl: ttl, release: redis.NewScript(releaseScript)}
}

// Acquire 는 key 를 잠근다. 성공하면 토큰과 true 를, 이미 잠겨 있으면 false 를 반환한다.
// 토큰은 Release 에 넘겨, 자기 잠금만 풀게 한다.
func (l *Lock) Acquire(ctx context.Context, key string) (token string, ok bool, err error) {
	token = newToken()
	ok, err = l.client.SetNX(ctx, key, token, l.ttl).Result()
	if err != nil {
		return "", false, err
	}
	return token, ok, nil
}

// Release 는 token 이 자기 것일 때만 잠금을 푼다.
func (l *Lock) Release(ctx context.Context, key, token string) error {
	return l.release.Run(ctx, l.client, []string{key}, token).Err()
}

func newToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Nop 은 잠금이 필요 없는 단일 인스턴스용 잠금이다(항상 획득 성공).
// Redis 가 없을 때 같은 코드 경로를 쓰기 위한 것.
type Nop struct{}

func (Nop) Acquire(context.Context, string) (string, bool, error) { return "local", true, nil }
func (Nop) Release(context.Context, string, string) error         { return nil }
