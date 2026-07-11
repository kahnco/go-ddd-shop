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

// DispatchOutbox 가 미발행 이벤트를 잠그고 발행 콜백을 호출한 뒤 발행표시하는지,
// 그리고 이미 발행된 것은 다시 집지 않는지 본다.
func TestDispatchOutbox_잠그고_발행하고_표시한다(t *testing.T) {
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

	var seen []infra.OutboxMessage
	n, err := repo.DispatchOutbox(ctx, func(m infra.OutboxMessage) error {
		seen = append(seen, m)
		return nil
	})
	if err != nil {
		t.Fatalf("DispatchOutbox: %v", err)
	}
	if n != 1 || len(seen) != 1 || seen[0].EventName != "order.placed" {
		t.Fatalf("1건이 발행 콜백으로 넘어와야 하는데 n=%d seen=%d", n, len(seen))
	}

	// 두 번째 디스패치는 이미 발행됐으니 0건이어야 한다.
	n2, err := repo.DispatchOutbox(ctx, func(infra.OutboxMessage) error { return nil })
	if err != nil {
		t.Fatalf("DispatchOutbox 2: %v", err)
	}
	if n2 != 0 {
		t.Fatalf("이미 발행된 뒤엔 0건이어야 하는데 %d건", n2)
	}
}
