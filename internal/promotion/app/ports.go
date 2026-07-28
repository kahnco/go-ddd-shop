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
// 원자성(동시 응모의 직렬화)은 구현(인메모리 뮤텍스 / Postgres 행 잠금)이 책임진다.
type Repository interface {
	SeedEvent(ctx context.Context, e domain.Event) error

	// Enter 는 (eventID, userID) 에 대해:
	//   - 처음이면 다음 순번(count+1)을 배정해 기록하고 Already=false.
	//   - 이미 응모했으면 기존 순번을 그대로 반환하고 Already=true (순번 미소비 → 멱등).
	//   - 시작 전이면 domain.ErrNotStarted, 없는 이벤트면 domain.ErrEventNotFound.
	// now 는 시작 시각 판정에 쓰는 현재 시각(테스트에서 주입 가능).
	Enter(ctx context.Context, eventID, userID string, now time.Time) (Result, error)
}

// EventPublisher 는 당첨 확정 이벤트를 밖으로 내보내는 포트.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}
