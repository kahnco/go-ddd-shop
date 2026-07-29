package domain

import "time"

// 프로모션 컨텍스트가 발행하는 도메인 이벤트. 다른 컨텍스트(결제·알림)가 구독한다.

type DomainEvent interface {
	EventName() string
}

// WinnerDetermined — 정확히 TargetSeq 번째 응모자가 확정됐다.
// 다운스트림이 이어받아 상금 지급·당첨 알림을 처리한다.
//
// 한 이벤트의 당첨자는 정확히 하나뿐이므로, DedupID 는 event_id 만으로 결정적이다.
// 브로커의 at-least-once 전달로 이 이벤트가 두 번 오더라도, 다운스트림은 DedupID 로
// 멱등하게 처리해 상금을 두 번 주지 않는다.
type WinnerDetermined struct {
	EventID      string    `json:"event_id"`
	UserID       string    `json:"user_id"`
	Seq          int       `json:"seq"`
	DeterminedAt time.Time `json:"determined_at"`
}

func (WinnerDetermined) EventName() string { return "winner_determined" }

// DedupID 는 다운스트림 멱등 처리를 위한 결정적 식별자다.
func (w WinnerDetermined) DedupID() string { return "promotion-winner:" + w.EventID }

// EventClosed — 이벤트가 종료됐다(당첨 확정 후 종료 배치가 발행).
// 다운스트림이 낙첨자 알림·정산·집계 마감 등을 이어받을 수 있다.
type EventClosed struct {
	EventID      string    `json:"event_id"`
	WinnerUserID string    `json:"winner_user_id"`
	TotalEntries int       `json:"total_entries"`
	ClosedAt     time.Time `json:"closed_at"`
}

func (EventClosed) EventName() string { return "event_closed" }

// DedupID 는 이벤트당 종료가 하나뿐이므로 event_id 만으로 결정적이다.
func (e EventClosed) DedupID() string { return "promotion-closed:" + e.EventID }
