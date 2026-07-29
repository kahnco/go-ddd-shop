package app

import (
	"context"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// Service 는 응모 유스케이스다.
// 순번 배정·당첨 확정·당첨 이벤트의 아웃박스 적재는 모두 Repository 가 한 트랜잭션에서 처리한다.
// (당첨 통지의 발행은 아웃박스 릴레이가 맡는다 — 커밋됐으면 반드시 발행된다.)
type Service struct {
	repo  Repository
	clock func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, clock: time.Now}
}

// WithClock 은 테스트에서 현재 시각을 고정하기 위한 옵션이다.
func (s *Service) WithClock(clock func() time.Time) *Service {
	s.clock = clock
	return s
}

func (s *Service) SeedEvent(ctx context.Context, e domain.Event) error {
	return s.repo.SeedEvent(ctx, e)
}

// Enter 는 사용자를 이벤트에 응모시킨다.
func (s *Service) Enter(ctx context.Context, eventID, userID string) (Result, error) {
	t0 := time.Now()
	res, err := s.repo.Enter(ctx, eventID, userID, s.clock())

	result := "assigned"
	switch {
	case err != nil:
		result = "rejected"
	case res.Already:
		result = "already"
	}
	telemetry.RecordPromotionEntry(eventID, result, res.Seq, time.Since(t0).Seconds())
	if err == nil && res.Winner && !res.Already {
		telemetry.RecordPromotionWinner(eventID)
	}
	return res, err
}

// EntryOf 는 사용자의 응모 상태를 조회한다(큐 모드).
func (s *Service) EntryOf(ctx context.Context, eventID, userID string) (Result, bool, error) {
	return s.repo.EntryOf(ctx, eventID, userID)
}

// MarkClosed 는 당첨이 확정된 이벤트를 종료 처리한다(멱등).
func (s *Service) MarkClosed(ctx context.Context, eventID string) (bool, error) {
	closed, err := s.repo.MarkClosed(ctx, eventID)
	if closed {
		telemetry.RecordPromotionClosed(eventID)
	}
	return closed, err
}
