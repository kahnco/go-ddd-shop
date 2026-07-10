package infra_test

import (
	"context"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
)

// 주문을 저장하면, 그 도메인 이벤트가 같은 트랜잭션으로 아웃박스에 적재되는지 본다.
// 그리고 발행표시 후에는 더 이상 조회되지 않는지(재발행 방지) 확인한다.
func TestOutbox_저장하면_이벤트가_아웃박스에_적재되고_발행표시된다(t *testing.T) {
	if testing.Short() {
		t.Skip("도커가 필요한 통합 테스트 — short 모드에서 건너뜀")
	}
	ctx := context.Background()
	repo, err := infra.NewPostgresOrderRepository(ctx, startPostgres(t))
	if err != nil {
		t.Fatalf("repo 생성: %v", err)
	}
	defer repo.Close()

	// 주문 저장 → PlaceOrder 가 낸 OrderPlaced 가 아웃박스에 들어가야 한다.
	if err := repo.Save(ctx, sampleOrder(t)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	msgs, err := repo.FetchOutbox(ctx, 10)
	if err != nil {
		t.Fatalf("FetchOutbox: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("아웃박스에 이벤트 1건 있어야 하는데 %d건", len(msgs))
	}
	if msgs[0].Subject != "ordering.order.placed" || msgs[0].EventName != "order.placed" {
		t.Fatalf("subject/event 불일치: %+v", msgs[0])
	}

	// 발행표시하면 미발행 목록에서 빠져야 한다.
	if err := repo.MarkOutboxPublished(ctx, []int64{msgs[0].ID}); err != nil {
		t.Fatalf("MarkOutboxPublished: %v", err)
	}
	after, _ := repo.FetchOutbox(ctx, 10)
	if len(after) != 0 {
		t.Fatalf("발행표시 후엔 미발행이 없어야 하는데 %d건", len(after))
	}
}
