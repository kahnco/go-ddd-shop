package infra

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// OutboxStore 는 릴레이가 필요로 하는 아웃박스 접근(가져오기·발행표시)만 추린 포트.
type OutboxStore interface {
	FetchOutbox(ctx context.Context, limit int) ([]OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, ids []int64) error
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
	msgs, err := r.store.FetchOutbox(ctx, 100)
	if err != nil {
		r.log.Error("아웃박스 조회 실패", "err", err)
		return
	}

	var published []int64
	for _, m := range msgs {
		env := eventbus.Envelope{
			ID:   strconv.FormatInt(m.ID, 10), // 재전송돼도 같은 ID → 소비자가 중복 제거 가능
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
		if err := r.bus.Publish(m.Subject, env); err != nil {
			r.log.Error("아웃박스 발행 실패(다음 주기에 재시도)", "id", m.ID, "err", err)
			break // 순서를 지키려 여기서 멈추고 다음 주기에 이어서
		}
		published = append(published, m.ID)
	}

	if err := r.store.MarkOutboxPublished(ctx, published); err != nil {
		// 발행은 됐지만 표시에 실패 → 다음 주기에 같은 이벤트를 다시 보낸다.
		// 그래서 소비자의 멱등성이 반드시 필요하다(at-least-once 의 대가).
		r.log.Error("아웃박스 발행표시 실패(중복 전송 가능)", "err", err)
	}
}
