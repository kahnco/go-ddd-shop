package infra_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// 정확히 N번째 당첨의 핵심 성질을, 동시에 몰린 응모로 검증한다(-race 로 돌린다):
//   - 순번이 빈틈·중복 없이 1..N 로 배정되고
//   - 당첨 순번(target)은 정확히 한 명에게만 간다.
func Test동시응모_순번은_빈틈없고_당첨은_정확히_하나(t *testing.T) {
	const N = 2000
	const target = 1000

	repo := infra.NewMemoryRepo()
	seed(t, repo, "ev", target)
	svc := app.NewService(repo)

	results := make([]app.Result, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := svc.Enter(context.Background(), "ev", "u-"+strconv.Itoa(i))
			if err != nil {
				t.Errorf("Enter: %v", err)
				return
			}
			results[i] = res
		}(i)
	}
	wg.Wait()

	// 1) 순번이 정확히 1..N 의 순열(빈틈·중복 없음)
	seen := make([]bool, N+1)
	winners := 0
	for _, r := range results {
		if r.Seq < 1 || r.Seq > N {
			t.Fatalf("순번 범위 벗어남: %d", r.Seq)
		}
		if seen[r.Seq] {
			t.Fatalf("순번 중복: %d", r.Seq)
		}
		seen[r.Seq] = true
		if r.Winner {
			winners++
			if r.Seq != target {
				t.Fatalf("당첨 순번이 %d (target=%d)", r.Seq, target)
			}
		}
	}
	for s := 1; s <= N; s++ {
		if !seen[s] {
			t.Fatalf("순번 빠짐(빈틈): %d", s)
		}
	}
	// 2) 당첨은 정확히 한 명
	if winners != 1 {
		t.Fatalf("당첨자 수 = %d (정확히 1이어야)", winners)
	}
	// 3) 저장소에 당첨자가 확정돼 있다
	if _, ok := repo.WinnerOf("ev"); !ok {
		t.Fatalf("당첨자가 저장소에 확정되지 않음")
	}
}

// 같은 사용자가 동시에 여러 번 응모해도 순번은 한 번만 소비된다(멱등).
func Test같은사용자_동시재응모는_순번을_하나만_쓴다(t *testing.T) {
	repo := infra.NewMemoryRepo()
	seed(t, repo, "ev", 1000)
	svc := app.NewService(repo)

	const tries = 500
	seqs := make(map[int]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < tries; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := svc.Enter(context.Background(), "ev", "same-user")
			if err != nil {
				t.Errorf("Enter: %v", err)
				return
			}
			mu.Lock()
			seqs[res.Seq] = struct{}{}
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(seqs) != 1 {
		t.Fatalf("같은 사용자에 서로 다른 순번 %d개 배정됨(1이어야)", len(seqs))
	}
}

func seed(t *testing.T, repo *infra.MemoryRepo, id string, target int) {
	t.Helper()
	if err := repo.SeedEvent(context.Background(), domain.Event{
		ID: id, TargetSeq: target, StartsAt: time.Unix(0, 0),
	}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
}
