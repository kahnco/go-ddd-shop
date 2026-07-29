package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// 당첨이 확정되면 아웃박스에 WinnerDetermined 가 적재되고, 릴레이가 한 번 발행한 뒤엔
// 다시 발행하지 않는다(발행 표시). 승리가 나기 전엔 아무것도 발행하지 않는다.
func TestOutbox_당첨이후에만_발행되고_한번뿐(t *testing.T) {
	repo := infra.NewMemoryRepo()
	_ = repo.SeedEvent(context.Background(), domain.Event{ID: "ev", TargetSeq: 2, StartsAt: time.Unix(0, 0)})
	svc := app.NewService(repo)

	var published []infra.OutboxMessage
	drain := func() (int, error) {
		return repo.DispatchOutbox(context.Background(), func(m infra.OutboxMessage) error {
			published = append(published, m)
			return nil
		})
	}

	// 아직 당첨 없음 → 발행할 것 없음
	if _, _ = svc.Enter(context.Background(), "ev", "alice"); len(published) != 0 {
		n, _ := drain()
		if n != 0 {
			t.Fatalf("당첨 전엔 발행이 없어야: %d", n)
		}
	}

	// 2번째 = 당첨 → 아웃박스 적재
	if r, _ := svc.Enter(context.Background(), "ev", "bob"); !r.Winner {
		t.Fatalf("bob 는 당첨이어야")
	}

	n, err := drain()
	if err != nil {
		t.Fatalf("drain: %v", err)
	}
	if n != 1 || len(published) != 1 {
		t.Fatalf("당첨 이벤트는 한 번 발행돼야: n=%d published=%d", n, len(published))
	}

	// 발행 내용 확인
	m := published[0]
	if m.Subject != "promotion.winner_determined" || m.DedupID != "promotion-winner:ev" {
		t.Fatalf("발행 봉투 이상: subject=%s dedup=%s", m.Subject, m.DedupID)
	}
	var w domain.WinnerDetermined
	if err := json.Unmarshal(m.Payload, &w); err != nil || w.UserID != "bob" || w.Seq != 2 {
		t.Fatalf("페이로드 이상: %+v err=%v", w, err)
	}

	// 다시 비워도 발행할 것 없음(이미 발행 표시)
	if n2, _ := drain(); n2 != 0 {
		t.Fatalf("이미 발행한 건 다시 발행하지 않아야: %d", n2)
	}
}
