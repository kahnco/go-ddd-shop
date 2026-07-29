package infra_test

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// 진짜 PostgreSQL(testcontainers)에 붙여, 여러 커넥션(= 여러 replica 관점)이 동시에
// 응모해도 순번이 빈틈없이 배정되고 당첨이 정확히 하나임을 검증한다.
func startPostgres(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("shop"),
		postgres.WithUsername("shop"),
		postgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("postgres 컨테이너 기동: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("연결 문자열: %v", err)
	}
	return dsn
}

func TestPostgres_동시응모_gapless_당첨정확히하나(t *testing.T) {
	ctx := context.Background()
	repo, err := infra.NewPostgresRepo(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("NewPostgresRepo: %v", err)
	}
	defer repo.Close()

	const N = 400
	const target = 200
	if err := repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: target, StartsAt: time.Unix(0, 0)}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}

	seqs := make([]int, N)
	winners := make([]bool, N)
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := repo.Enter(ctx, "ev", "u-"+strconv.Itoa(i), time.Now())
			if err != nil {
				t.Errorf("Enter: %v", err)
				return
			}
			seqs[i] = res.Seq
			winners[i] = res.Winner
		}(i)
	}
	wg.Wait()

	seen := make([]bool, N+1)
	winCount := 0
	for i, s := range seqs {
		if s < 1 || s > N || seen[s] {
			t.Fatalf("순번 이상(빈틈/중복/범위): %d", s)
		}
		seen[s] = true
		if winners[i] {
			winCount++
			if s != target {
				t.Fatalf("당첨 순번 %d (target=%d)", s, target)
			}
		}
	}
	for s := 1; s <= N; s++ {
		if !seen[s] {
			t.Fatalf("순번 빠짐(빈틈): %d", s)
		}
	}
	if winCount != 1 {
		t.Fatalf("당첨자 수=%d (정확히 1)", winCount)
	}
	if w, ok, _ := repo.WinnerOf(ctx, "ev"); !ok || w == "" {
		t.Fatalf("당첨자가 이벤트 행에 확정되지 않음")
	}
}

func TestPostgres_시작전응모는_순번을_소비하지않는다(t *testing.T) {
	ctx := context.Background()
	repo, err := infra.NewPostgresRepo(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("NewPostgresRepo: %v", err)
	}
	defer repo.Close()

	future := time.Now().Add(time.Hour)
	if err := repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: 1000, StartsAt: future}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	// 시작 전 응모 여러 번 — 전부 거부
	for i := 0; i < 5; i++ {
		if _, err := repo.Enter(ctx, "ev", "u-"+strconv.Itoa(i), future.Add(-time.Minute)); err != domain.ErrNotStarted {
			t.Fatalf("시작 전 응모는 ErrNotStarted 여야: %v", err)
		}
	}
	// 시작 후 첫 응모가 1번이어야(거부가 순번을 새게 하지 않음)
	res, err := repo.Enter(ctx, "ev", "first", future.Add(time.Minute))
	if err != nil {
		t.Fatalf("시작 후 응모: %v", err)
	}
	if res.Seq != 1 {
		t.Fatalf("첫 순번이 %d (1이어야) — 빈틈 발생", res.Seq)
	}
}

func TestPostgres_당첨이_아웃박스에_적재되고_한번_디스패치된다(t *testing.T) {
	ctx := context.Background()
	repo, err := infra.NewPostgresRepo(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("NewPostgresRepo: %v", err)
	}
	defer repo.Close()

	if err := repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: 2, StartsAt: time.Unix(0, 0)}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	_, _ = repo.Enter(ctx, "ev", "alice", time.Now()) // seq 1
	r, _ := repo.Enter(ctx, "ev", "bob", time.Now())   // seq 2 = 당첨
	if !r.Winner {
		t.Fatalf("bob 는 당첨이어야")
	}

	var published []infra.OutboxMessage
	n, err := repo.DispatchOutbox(ctx, func(m infra.OutboxMessage) error {
		published = append(published, m)
		return nil
	})
	if err != nil {
		t.Fatalf("DispatchOutbox: %v", err)
	}
	if n != 1 || len(published) != 1 || published[0].DedupID != "promotion-winner:ev" {
		t.Fatalf("당첨 이벤트 1건이 발행돼야: n=%d, %+v", n, published)
	}
	// 재디스패치 → 이미 발행 표시라 0건
	if n2, _ := repo.DispatchOutbox(ctx, func(infra.OutboxMessage) error { return nil }); n2 != 0 {
		t.Fatalf("이미 발행한 건 다시 발행하지 않아야: %d", n2)
	}
}

func TestPostgres_같은사용자_재응모는_멱등(t *testing.T) {
	ctx := context.Background()
	repo, err := infra.NewPostgresRepo(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("NewPostgresRepo: %v", err)
	}
	defer repo.Close()

	if err := repo.SeedEvent(ctx, domain.Event{ID: "ev", TargetSeq: 1000, StartsAt: time.Unix(0, 0)}); err != nil {
		t.Fatalf("SeedEvent: %v", err)
	}
	first, _ := repo.Enter(ctx, "ev", "same", time.Now())
	second, _ := repo.Enter(ctx, "ev", "same", time.Now())
	if first.Seq != second.Seq {
		t.Fatalf("같은 사용자 재응모 순번이 다름: %d vs %d", first.Seq, second.Seq)
	}
	if !second.Already {
		t.Fatalf("재응모는 Already 여야")
	}
	// 다른 사용자는 2번(같은 사용자가 순번을 하나만 썼으니)
	other, _ := repo.Enter(ctx, "ev", "other", time.Now())
	if other.Seq != 2 {
		t.Fatalf("다음 사용자 순번이 %d (2여야)", other.Seq)
	}
}
