package infra

import (
	"context"
	"log/slog"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

// OutboxMessage 는 아웃박스에 적재된 미발행 이벤트 한 건이다.
// DedupID 는 결정적 식별자 — 봉투 ID 로 실려, 다운스트림이 중복 수신을 걸러낸다.
type OutboxMessage struct {
	ID        int64
	Subject   string
	EventName string
	Payload   []byte
	DedupID   string
}

// OutboxStore 는 릴레이가 필요로 하는 아웃박스 디스패치 포트.
// 잠금·발행·표시를 한 트랜잭션으로 묶어, 아웃박스에 적재된 이벤트를 브로커로 흘린다.
type OutboxStore interface {
	DispatchOutbox(ctx context.Context, publish func(OutboxMessage) error) (int, error)
}

// OutboxRelay 는 아웃박스에 쌓인 당첨 이벤트를 주기적으로 읽어 브로커로 발행한다.
// 응모와 같은 트랜잭션에서 아웃박스에 적재되므로("커밋됐으면 반드시 발행"),
// 커밋 후 바로 발행하다 유실되는 문제를 없앤다(4·9편의 아웃박스 패턴).
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
			if _, err := r.store.DispatchOutbox(ctx, r.publish); err != nil {
				r.log.Error("아웃박스 디스패치 실패(다음 주기 재시도)", "err", err)
			}
		}
	}
}

func (r *OutboxRelay) publish(m OutboxMessage) error {
	// 봉투 ID = DedupID(결정적) → JetStream 중복 제거 + 소비자 멱등의 열쇠.
	env := eventbus.Envelope{ID: m.DedupID, Name: m.EventName, Data: m.Payload}
	return r.bus.Publish(m.Subject, env)
}
