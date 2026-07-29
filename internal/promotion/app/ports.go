package app

import (
	"context"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// Result 는 한 번의 응모 결과다.
type Result struct {
	Seq     int  // 배정된 순번(빈틈없음)
	Winner  bool // Seq 가 당첨 순번인지
	Already bool // 이미 응모했던 사용자였는지(이번엔 순번을 소비하지 않음)
}

// Repository 는 응모를 멱등하게 기록하고 빈틈없는 순번을 배정하는 포트.
// 원자성(동시 응모의 직렬화)과, 당첨 시 WinnerDetermined 를 아웃박스에 같은 트랜잭션으로
// 적재하는 일까지 구현(인메모리 뮤텍스 / Postgres 행 잠금)이 책임진다.
type Repository interface {
	SeedEvent(ctx context.Context, e domain.Event) error

	// Enter 는 (eventID, userID) 에 대해:
	//   - 처음이면 다음 순번(count+1)을 배정해 기록하고 Already=false.
	//   - 이미 응모했으면 기존 순번을 그대로 반환하고 Already=true (순번 미소비 → 멱등).
	//   - 시작 전이면 domain.ErrNotStarted, 없는 이벤트면 domain.ErrEventNotFound.
	//   - 당첨이 이번에 처음 확정되면, WinnerDetermined 를 아웃박스에 같은 트랜잭션으로 적재한다.
	Enter(ctx context.Context, eventID, userID string, now time.Time) (Result, error)

	// EntryOf 는 사용자의 현재 응모 상태를 조회한다(큐 모드의 상태 확인용).
	// 아직 응모가 기록되지 않았으면 found=false.
	EntryOf(ctx context.Context, eventID, userID string) (res Result, found bool, err error)
}

// EntryQueue 는 응모 요청을 브로커로 흘려보내는 포트(큐 완충 모드).
// 접수는 빠르게 받고, 순번 배정은 뒤에서 순차 소비자가 한다.
type EntryQueue interface {
	Enqueue(ctx context.Context, eventID, userID, requestID string) error
}
