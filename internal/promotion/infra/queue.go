package infra

import (
	"context"
	"log/slog"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/idempotency"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
)

// EntryRequestedSubject — 접수된 응모 요청이 흐르는 주제.
const EntryRequestedSubject = "promotion.entry_requested"

// entryRequestedPayload 는 접수 시점의 응모 요청. RequestedAt(접수 시각)을 실어,
// 소비자가 도착 순서대로 순번을 매길 수 있게 한다(공정성).
type entryRequestedPayload struct {
	EventID     string    `json:"event_id"`
	UserID      string    `json:"user_id"`
	RequestID   string    `json:"request_id"`
	RequestedAt time.Time `json:"requested_at"`
}

// NatsEntryQueue 는 app.EntryQueue 를 NATS 발행으로 구현한다(큐 완충 모드의 접수).
// 접수는 여기서 빠르게 끝나고, 순번 배정은 뒤에서 EntryRequestedConsumer 가 한다.
type NatsEntryQueue struct {
	bus *eventbus.Bus
}

func NewNatsEntryQueue(bus *eventbus.Bus) *NatsEntryQueue { return &NatsEntryQueue{bus: bus} }

func (q *NatsEntryQueue) Enqueue(ctx context.Context, eventID, userID, requestID string) error {
	env, err := eventbus.NewEnvelope("entry_requested", entryRequestedPayload{
		EventID:     eventID,
		UserID:      userID,
		RequestID:   requestID,
		RequestedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	env.ID = requestID // 소비자 멱등의 열쇠(재전송돼도 같은 요청)
	env.Meta = telemetry.MetaFromContext(ctx)
	return q.bus.Publish(EntryRequestedSubject, env)
}

var _ app.EntryQueue = (*NatsEntryQueue)(nil)

// EntryRequestedConsumer 는 접수된 응모 요청을 꺼내 실제 순번 배정(Enter)을 수행하는 소비자.
// 단일 구독(구독당 goroutine 하나)이라 접수 순서대로 처리된다.
// request_id 로 중복을 거르지만, 순번 배정 자체도 (event,user) 로 멱등이라 이중 안전망이다.
type EntryRequestedConsumer struct {
	svc   *app.Service
	guard *idempotency.Guard
	log   *slog.Logger
}

func NewEntryRequestedConsumer(svc *app.Service, log *slog.Logger) *EntryRequestedConsumer {
	return &EntryRequestedConsumer{svc: svc, guard: idempotency.NewGuard(), log: log}
}

func (c *EntryRequestedConsumer) Handle(env eventbus.Envelope) error {
	ctx := telemetry.ContextFromMeta(context.Background(), env.Meta)
	var p entryRequestedPayload
	if err := env.Into(&p); err != nil {
		c.log.Error("entry_requested 디코딩 실패", "err", err)
		return err
	}
	return c.guard.Do(env.ID, func() error {
		res, err := c.svc.Enter(ctx, p.EventID, p.UserID)
		if err != nil {
			c.log.Error("응모 처리 실패", "event", p.EventID, "user", p.UserID, "err", err)
			return err
		}
		if res.Winner && !res.Already {
			c.log.Info("당첨 확정", "event", p.EventID, "user", p.UserID, "seq", res.Seq)
		}
		return nil
	})
}
