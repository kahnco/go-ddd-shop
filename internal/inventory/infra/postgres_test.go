package infra_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
	"github.com/kahnco/go-ddd-shop/internal/inventory/infra"
)

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

// 실제 PostgreSQL 에서, 같은 상품에 동시 차감이 몰려도 FOR UPDATE 행 잠금 덕에
// 재고가 정확히 유지되는지 본다(여러 replica 를 흉내 낸 동시 Update).
func TestPostgresStore_동시_차감이_몰려도_재고가_정확하다(t *testing.T) {
	if testing.Short() {
		t.Skip("도커가 필요한 통합 테스트 — short 모드에서 건너뜀")
	}
	ctx := context.Background()
	store, err := infra.NewPostgresStore(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("store 생성: %v", err)
	}
	defer store.Close()
	if err := store.Seed(ctx, "prod-A", 1000); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = store.Update(ctx, "prod-A", func(si *domain.StockItem) error {
				return si.Reserve(1)
			})
		}()
	}
	wg.Wait()

	item, _ := store.FindByProduct(ctx, "prod-A")
	if item.Available() != 0 {
		t.Fatalf("동시 차감 1000건 후 재고 = 0 이어야 하는데 %d (레이스 발생)", item.Available())
	}
}
