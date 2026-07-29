package infra

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/idempotency"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// isTerminal 은 재시도해도 결과가 바뀌지 않는 영구 거부인지 판단한다.
func isTerminal(err error) bool {
	return errors.Is(err, domain.ErrClosed) ||
		errors.Is(err, domain.ErrNotStarted) ||
		errors.Is(err, domain.ErrEventNotFound)
}

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
	svc      *app.Service
	guard    *idempotency.Guard
	notifier EntryNotifier // 선택 — 있으면 배정 결과를 통지(로컬 허브 또는 브로드캐스트)
	log      *slog.Logger
}

func NewEntryRequestedConsumer(svc *app.Service, log *slog.Logger) *EntryRequestedConsumer {
	return &EntryRequestedConsumer{svc: svc, guard: idempotency.NewGuard(), log: log}
}

// WithNotifier 는 처리 결과를 실시간으로 알릴 통지자를 연결한다(SSE 허브·브로드캐스트).
func (c *EntryRequestedConsumer) WithNotifier(n EntryNotifier) *EntryRequestedConsumer {
	c.notifier = n
	return c
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
			// 종료·시작전·없는 이벤트는 "영구 거부"다 — 재시도해도 결과가 같으니
			// 로그만 남기고 ack(드롭)한다. 재시도·DLQ 는 일시적 오류(DB 장애 등)에만.
			if isTerminal(err) {
				c.log.Warn("응모 영구 거부(드롭)", "event", p.EventID, "user", p.UserID, "reason", err)
				return nil
			}
			c.log.Error("응모 처리 실패(재시도)", "event", p.EventID, "user", p.UserID, "err", err)
			return err
		}
		if c.notifier != nil {
			c.notifier.NotifyEntry(p.EventID, p.UserID, res) // 배정 결과 통지(로컬/브로드캐스트)
		}
		if res.Winner && !res.Already {
			c.log.Info("당첨 확정", "event", p.EventID, "user", p.UserID, "seq", res.Seq)
		}
		return nil
	})
}
