package infra

import (
	"log/slog"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
)

// EntryAssignedSubject — 배정 결과를 모든 인스턴스로 팬아웃하는 주제(영속 아님).
const EntryAssignedSubject = "promotion.notify.entry"

// EntryNotifier 는 배정 결과를 통지하는 포트. 로컬 허브(단일 인스턴스) 또는
// 브로드캐스트(다중 인스턴스) 중 무엇이든 될 수 있다.
type EntryNotifier interface {
	NotifyEntry(eventID, userID string, res app.Result)
}

// Hub 는 로컬 EntryNotifier 다(같은 인스턴스의 구독자에게 직접 push).
var _ EntryNotifier = (*Hub)(nil)

type entryAssignedPayload struct {
	EventID string `json:"event_id"`
	UserID  string `json:"user_id"`
	Seq     int    `json:"seq"`
	Winner  bool   `json:"winner"`
	Already bool   `json:"already"`
}

// BroadcastNotifier 는 배정 결과를 core NATS 로 팬아웃한다.
// 소비자는 한 인스턴스에서만 돌지만, SSE 연결은 어느 인스턴스에 붙어 있을지 모른다.
// 그래서 결과를 모든 인스턴스로 뿌리고, 각자 자기 허브로 push 하게 한다.
type BroadcastNotifier struct {
	bus *eventbus.Bus
}

func NewBroadcastNotifier(bus *eventbus.Bus) *BroadcastNotifier { return &BroadcastNotifier{bus: bus} }

func (n *BroadcastNotifier) NotifyEntry(eventID, userID string, res app.Result) {
	env, err := eventbus.NewEnvelope("entry_assigned", entryAssignedPayload{
		EventID: eventID, UserID: userID, Seq: res.Seq, Winner: res.Winner, Already: res.Already,
	})
	if err != nil {
		return
	}
	_ = n.bus.Broadcast(EntryAssignedSubject, env)
}

var _ EntryNotifier = (*BroadcastNotifier)(nil)

// NotifyBridge 는 팬아웃된 통지를 받아 로컬 허브로 push 한다(인스턴스마다 하나 실행).
// 이 덕에 소비자가 어느 인스턴스에서 배정했든, SSE 가 붙은 인스턴스가 결과를 흘려보낸다.
type NotifyBridge struct {
	bus *eventbus.Bus
	hub *Hub
	log *slog.Logger
}

func NewNotifyBridge(bus *eventbus.Bus, hub *Hub, log *slog.Logger) *NotifyBridge {
	return &NotifyBridge{bus: bus, hub: hub, log: log}
}

func (br *NotifyBridge) Start() error {
	return br.bus.SubscribeBroadcast(EntryAssignedSubject, func(env eventbus.Envelope) error {
		var p entryAssignedPayload
		if err := env.Into(&p); err != nil {
			return err
		}
		br.hub.NotifyEntry(p.EventID, p.UserID, app.Result{Seq: p.Seq, Winner: p.Winner, Already: p.Already})
		return nil
	})
}
