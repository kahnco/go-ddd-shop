package distlock_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kahnco/go-ddd-shop/internal/platform/distlock"
)

func startRedis(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("redis 컨테이너 기동: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "6379")
	client := redis.NewClient(&redis.Options{Addr: host + ":" + port.Port()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestLock_한번에_하나만_잡고_해제하면_다시_잡힌다(t *testing.T) {
	ctx := context.Background()
	l := distlock.New(startRedis(t), 30*time.Second)

	tok1, ok1, err := l.Acquire(ctx, "close:ev")
	if err != nil || !ok1 {
		t.Fatalf("첫 획득 성공해야: ok=%v err=%v", ok1, err)
	}
	// 이미 잡혀 있음 → 두 번째는 실패
	if _, ok2, _ := l.Acquire(ctx, "close:ev"); ok2 {
		t.Fatalf("이미 잠긴 키는 획득 실패해야")
	}
	// 해제 후 다시 획득 가능
	if err := l.Release(ctx, "close:ev", tok1); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, ok3, _ := l.Acquire(ctx, "close:ev"); !ok3 {
		t.Fatalf("해제 후엔 다시 획득돼야")
	}
}

func TestLock_남의_토큰으로는_풀_수_없다(t *testing.T) {
	ctx := context.Background()
	l := distlock.New(startRedis(t), 30*time.Second)

	if _, ok, _ := l.Acquire(ctx, "k"); !ok {
		t.Fatal("획득 실패")
	}
	// 엉뚱한 토큰으로 Release → 잠금 유지
	_ = l.Release(ctx, "k", "wrong-token")
	if _, ok, _ := l.Acquire(ctx, "k"); ok {
		t.Fatalf("남의 토큰으로 풀리면 안 됨(여전히 잠겨 있어야)")
	}
}
