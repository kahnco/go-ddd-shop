package infra

import (
	"context"
	"log/slog"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
)

// Locker 는 종료 배치가 필요로 하는 분산 잠금 포트(distlock.Lock / distlock.Nop 이 구현).
type Locker interface {
	Acquire(ctx context.Context, key string) (token string, ok bool, err error)
	Release(ctx context.Context, key, token string) error
}

// Closer 는 당첨이 확정된 이벤트를 "정확히 한 인스턴스가 한 번" 종료 처리한다.
// 여러 replica 가 각자 주기적으로 시도하지만, 분산 잠금으로 실제 배치는 하나만 돈다.
// (MarkClosed 자체도 멱등이라 이중 안전망 — 잠금은 무거운 배치를 중복 실행하지 않게 한다.)
type Closer struct {
	svc      *app.Service
	lock     Locker
	events   []string
	interval time.Duration
	log      *slog.Logger
}

func NewCloser(svc *app.Service, lock Locker, interval time.Duration, log *slog.Logger) *Closer {
	return &Closer{svc: svc, lock: lock, interval: interval, log: log}
}

// Watch 는 종료 여부를 주기적으로 확인할 이벤트를 등록한다.
func (c *Closer) Watch(eventID string) *Closer {
	c.events = append(c.events, eventID)
	return c
}

// Run 은 ctx 가 끝날 때까지 주기적으로 각 이벤트의 종료를 시도한다.
func (c *Closer) Run(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for _, ev := range c.events {
				c.tryClose(ctx, ev)
			}
		}
	}
}

func (c *Closer) tryClose(ctx context.Context, eventID string) {
	key := "promotion:close:" + eventID
	token, ok, err := c.lock.Acquire(ctx, key)
	if err != nil {
		c.log.Error("종료 잠금 획득 실패", "event", eventID, "err", err)
		return
	}
	if !ok {
		return // 다른 인스턴스가 종료 배치를 잡았다
	}
	defer func() { _ = c.lock.Release(ctx, key, token) }()

	closed, err := c.svc.MarkClosed(ctx, eventID)
	if err != nil {
		c.log.Error("종료 처리 실패", "event", eventID, "err", err)
		return
	}
	if closed {
		c.log.Info("이벤트 종료 배치 실행 — 이 인스턴스가 확정", "event", eventID)
	}
}
