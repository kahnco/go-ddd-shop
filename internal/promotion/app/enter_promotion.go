package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// Service 는 응모 유스케이스다.
// 순번 배정과 당첨 확정은 Repository 가 원자적으로 처리하고(정확성의 핵심),
// Service 는 당첨이 "이번 응모로" 처음 확정된 경우에만 WinnerDetermined 를 발행한다.
type Service struct {
	repo   Repository
	pub    EventPublisher
	clock  func() time.Time
	logger *slog.Logger
}

func NewService(repo Repository, pub EventPublisher) *Service {
	return &Service{repo: repo, pub: pub, clock: time.Now, logger: slog.Default()}
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
	res, err := s.repo.Enter(ctx, eventID, userID, s.clock())
	if err != nil {
		return Result{}, err
	}

	// 당첨이 "이번 응모로" 처음 확정됐을 때만 발행한다.
	// 멱등 재응모(Already)는 이미 발행됐으므로 다시 쏘지 않는다.
	if res.Winner && !res.Already && s.pub != nil {
		evt := domain.WinnerDetermined{
			EventID:      eventID,
			UserID:       userID,
			Seq:          res.Seq,
			DeterminedAt: s.clock(),
		}
		if err := s.pub.Publish(ctx, evt); err != nil {
			// 당첨은 저장소에 이미 확정(감사 가능)됐지만 발행이 실패한 경우다.
			// 커밋-후-발행의 한계 — 실서비스에서는 아웃박스로 응모와 같은 트랜잭션에서
			// 발행 레코드를 남겨 재전송해야 한다. 여기서는 로그로 남긴다.
			s.logger.Error("당첨 이벤트 발행 실패 — 재전송/아웃박스 필요",
				"event", eventID, "user", userID, "err", err)
		}
	}
	return res, nil
}
