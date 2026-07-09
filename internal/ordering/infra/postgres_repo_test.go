package infra_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
)

// 실제 PostgreSQL(testcontainers 로 띄운 진짜 컨테이너)에 대해 저장·재구성을 검증한다.
// 임베디드 NATS 와 같은 철학 — 목(mock)이 아니라 진짜에 붙여, SQL·매핑까지 확인한다.
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

func sampleOrder(t *testing.T) *domain.Order {
	t.Helper()
	qtyA, _ := domain.NewQuantity(2)
	priceA, _ := domain.NewMoney(1000)
	qtyB, _ := domain.NewQuantity(1)
	priceB, _ := domain.NewMoney(3000)
	o, err := domain.PlaceOrder("order-1", "cust-1", []domain.OrderLine{
		domain.NewOrderLine("prod-A", qtyA, priceA),
		domain.NewOrderLine("prod-B", qtyB, priceB),
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	return o
}

func TestPostgresRepo_저장하고_다시_읽으면_같은_주문(t *testing.T) {
	if testing.Short() {
		t.Skip("도커가 필요한 통합 테스트 — short 모드에서 건너뜀")
	}
	ctx := context.Background()
	repo, err := infra.NewPostgresOrderRepository(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("repo 생성: %v", err)
	}
	defer repo.Close()

	if err := repo.Save(ctx, sampleOrder(t)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, "order-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.CustomerID() != "cust-1" {
		t.Fatalf("고객 = cust-1 여야 하는데 %s", got.CustomerID())
	}
	if got.Total().Amount() != 5000 { // 항목에서 재계산된 총액
		t.Fatalf("총액 = 5000 여야 하는데 %d", got.Total().Amount())
	}
	if len(got.Lines()) != 2 {
		t.Fatalf("항목 2개여야 하는데 %d개", len(got.Lines()))
	}
	// 재구성된 주문은 이벤트를 다시 내지 않아야 한다(과거 사실의 복원이므로).
	if evs := got.PullEvents(); len(evs) != 0 {
		t.Fatalf("재구성 시 이벤트가 없어야 하는데 %d개", len(evs))
	}
}

func TestPostgresRepo_상태_변경이_저장된다(t *testing.T) {
	if testing.Short() {
		t.Skip("도커가 필요한 통합 테스트 — short 모드에서 건너뜀")
	}
	ctx := context.Background()
	repo, err := infra.NewPostgresOrderRepository(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("repo 생성: %v", err)
	}
	defer repo.Close()

	o := sampleOrder(t)
	_ = repo.Save(ctx, o)

	// 결제 처리 후 다시 저장(upsert) → 상태가 갱신돼야 한다.
	if err := o.MarkPaid(); err != nil {
		t.Fatalf("MarkPaid: %v", err)
	}
	if err := repo.Save(ctx, o); err != nil {
		t.Fatalf("재저장: %v", err)
	}

	got, _ := repo.FindByID(ctx, "order-1")
	if got.Status() != domain.StatusPaid {
		t.Fatalf("상태 = PAID 여야 하는데 %s", got.Status())
	}
}

func TestPostgresRepo_없는_주문은_ErrOrderNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("도커가 필요한 통합 테스트 — short 모드에서 건너뜀")
	}
	ctx := context.Background()
	repo, err := infra.NewPostgresOrderRepository(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("repo 생성: %v", err)
	}
	defer repo.Close()

	if _, err := repo.FindByID(ctx, "nope"); !errors.Is(err, domain.ErrOrderNotFound) {
		t.Fatalf("없는 주문은 ErrOrderNotFound 여야 하는데: %v", err)
	}
}
