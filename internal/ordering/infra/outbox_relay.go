package infra

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// OutboxStore 는 릴레이가 필요로 하는 아웃박스 디스패치 포트.
// 잠금·발행·표시를 한 트랜잭션으로 묶어(SKIP LOCKED), 여러 릴레이가 안전하게 병렬로 돈다.
type OutboxStore interface {
	DispatchOutbox(ctx context.Context, publish func(OutboxMessage) error) (int, error)
}

// OutboxRelay 는 아웃박스에 쌓인 이벤트를 주기적으로 읽어 브로커로 발행하는 디스패처.
// 아웃박스에 이벤트가 저장되는 것과 실제 발행을 분리해, 저장은 트랜잭션으로 확실히 하고
// 발행은 이 릴레이가 "될 때까지" 재시도하게 한다(at-least-once).
type OutboxRelay struct {
	store    OutboxStore
	bus      *eventbus.Bus
	interval time.Duration
	log      *slog.Logger
}

func NewOutboxRelay(store OutboxStore, bus *eventbus.Bus, interval time.Duration, log *slog.Logger) *OutboxRelay {
	return &OutboxRelay{store: store, bus: bus, interval: interval, log: log}
}

// Run 은 ctx 가 끝날 때까지 주기적으로 아웃박스를 비운다.
func (r *OutboxRelay) Run(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.dispatch(ctx)
		}
	}
}

func (r *OutboxRelay) dispatch(ctx context.Context) {
	// 잠금·발행·표시가 한 트랜잭션. SKIP LOCKED 로 다른 릴레이와 행이 겹치지 않는다.
	_, err := r.store.DispatchOutbox(ctx, r.publish)
	if err != nil {
		r.log.Error("아웃박스 디스패치 실패(다음 주기에 재시도)", "err", err)
	}
}

func (r *OutboxRelay) publish(m OutboxMessage) error {
	env := eventbus.Envelope{
		ID:   strconv.FormatInt(m.ID, 10), // 안정적 ID → JetStream Nats-Msg-Id 중복 제거 + 소비자 멱등
		Name: m.EventName,
		Data: m.Payload,
	}
	meta := map[string]string{}
	if m.CorrelationID != "" {
		meta[telemetry.MetaCorrelationID] = m.CorrelationID
	}
	if m.Traceparent != "" {
		meta["traceparent"] = m.Traceparent // 저장해 둔 trace 컨텍스트를 소비자에게 잇는다
	}
	if len(meta) > 0 {
		env.Meta = meta
	}
	return r.bus.Publish(m.Subject, env)
}
