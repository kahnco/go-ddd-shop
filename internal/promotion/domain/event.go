package domain

import "time"

// Event 는 "정확히 TargetSeq 번째로 응모한 사람이 당첨"인 프로모션 이벤트다.
// 예: TargetSeq=1000 이면, 1000번째로 유효하게 응모한 사용자가 상금을 받는다.
type Event struct {
	ID        string
	TargetSeq int       // 당첨 순번 (예: 1000)
	StartsAt  time.Time // 이 시각 이전 응모는 무효
}

// IsWinner 는 주어진 순번이 이 이벤트의 당첨 순번인지 판단한다.
func (e Event) IsWinner(seq int) bool { return seq == e.TargetSeq }

// Entry 는 한 사용자의 응모 기록이다.
// Seq 는 빈틈없이(gapless) 배정된 순번 — "정확히 N번째"의 근거가 된다.
type Entry struct {
	EventID   string
	UserID    string
	Seq       int
	CreatedAt time.Time
}
