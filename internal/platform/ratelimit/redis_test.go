package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kahnco/go-ddd-shop/internal/platform/ratelimit"
)

// 진짜 Redis(testcontainers)에 붙여, 토큰 버킷 Lua 가 원자적으로 동작하는지 검증한다.
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

func TestRedis_버스트까지_허용하고_초과는_거절하고_회복한다(t *testing.T) {
	ctx := context.Background()
	client := startRedis(t)

	now := time.Unix(1000, 0)
	l := ratelimit.NewRedis(client, 3, 1).WithClock(func() time.Time { return now })

	for i := 0; i < 3; i++ {
		ok, err := l.Allow(ctx, "u1")
		if err != nil || !ok {
			t.Fatalf("버스트 안(%d)은 허용돼야: ok=%v err=%v", i+1, ok, err)
		}
	}
	if ok, _ := l.Allow(ctx, "u1"); ok {
		t.Fatalf("버스트 초과는 거절돼야")
	}
	// 다른 키는 독립
	if ok, _ := l.Allow(ctx, "u2"); !ok {
		t.Fatalf("다른 키는 각자 버킷이어야")
	}
	// 1초 뒤 토큰 1개 회복
	now = now.Add(time.Second)
	if ok, _ := l.Allow(ctx, "u1"); !ok {
		t.Fatalf("1초 뒤엔 회복돼 허용돼야")
	}
}

// 두 리미터 인스턴스가 같은 Redis 를 보면 한도를 공유한다(분산 리밋의 핵심).
func TestRedis_인스턴스가_달라도_한도를_공유한다(t *testing.T) {
	ctx := context.Background()
	client := startRedis(t)
	now := time.Unix(2000, 0)

	a := ratelimit.NewRedis(client, 2, 1).WithClock(func() time.Time { return now })
	b := ratelimit.NewRedis(client, 2, 1).WithClock(func() time.Time { return now })

	if ok, _ := a.Allow(ctx, "shared"); !ok {
		t.Fatal("a 1회 허용")
	}
	if ok, _ := b.Allow(ctx, "shared"); !ok {
		t.Fatal("b 도 같은 버킷에서 2회째 허용")
	}
	// 버스트 2 소진 — 어느 인스턴스든 거절
	if ok, _ := a.Allow(ctx, "shared"); ok {
		t.Fatal("공유 한도 소진 후 a 는 거절돼야")
	}
	if ok, _ := b.Allow(ctx, "shared"); ok {
		t.Fatal("공유 한도 소진 후 b 도 거절돼야")
	}
}
