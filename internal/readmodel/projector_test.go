package readmodel

import (
	"io"
	"log/slog"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

func placedEnv(t *testing.T, orderID, customerID string, total int64) eventbus.Envelope {
	t.Helper()
	env, err := eventbus.NewEnvelope("order.placed", map[string]any{
		"order_id":    orderID,
		"customer_id": customerID,
		"total":       total,
		"items":       []map[string]any{{"product_id": "prod-A", "quantity": 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func statusEnv(t *testing.T, name, orderID string) eventbus.Envelope {
	t.Helper()
	env, err := eventbus.NewEnvelope(name, map[string]any{"order_id": orderID})
	if err != nil {
		t.Fatal(err)
	}
	return env
}

func newProjector() (*Projector, *MemoryStore) {
	store := NewMemoryStore()
	return NewProjector(store, slog.New(slog.NewTextHandler(io.Discard, nil))), store
}

func TestProjector_주문_생성후_배송까지_상태가_반영된다(t *testing.T) {
	p, store := newProjector()

	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 5000))
	_ = p.Handle(statusEnv(t, "order.paid", "order-1"))
	_ = p.Handle(statusEnv(t, "order.confirmed", "order-1"))
	_ = p.Handle(statusEnv(t, "order.shipped", "order-1"))

	v, ok := store.Get("order-1")
	if !ok {
		t.Fatal("주문 뷰가 있어야 함")
	}
	if v.Status != "SHIPPED" || v.Total != 5000 || v.CustomerID != "cust-1" {
		t.Fatalf("뷰 불일치: %+v", v)
	}
}

func TestProjector_회원별_주문목록을_돌려준다(t *testing.T) {
	p, store := newProjector()
	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 1000))
	_ = p.Handle(placedEnv(t, "order-2", "cust-1", 2000))
	_ = p.Handle(placedEnv(t, "order-3", "cust-2", 3000))

	mine := store.ByCustomer("cust-1")
	if len(mine) != 2 {
		t.Fatalf("cust-1 주문 2건이어야 하는데 %d건", len(mine))
	}
	if mine[0].OrderID != "order-1" || mine[1].OrderID != "order-2" {
		t.Fatalf("주문 ID 순 정렬돼야 하는데 %+v", mine)
	}
}

func TestProjector_집계_상태별건수와_매출(t *testing.T) {
	p, store := newProjector()
	// 주문 2건 배송 완료, 1건 취소.
	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 1000))
	_ = p.Handle(statusEnv(t, "order.shipped", "order-1"))
	_ = p.Handle(placedEnv(t, "order-2", "cust-1", 2000))
	_ = p.Handle(statusEnv(t, "order.shipped", "order-2"))
	_ = p.Handle(placedEnv(t, "order-3", "cust-2", 5000))
	_ = p.Handle(statusEnv(t, "order.cancelled", "order-3"))

	s := store.Stats()
	if s.OrderCount != 3 {
		t.Fatalf("전체 주문 3건이어야 하는데 %d", s.OrderCount)
	}
	if s.Counts["SHIPPED"] != 2 || s.Counts["CANCELLED"] != 1 {
		t.Fatalf("상태별 건수 불일치: %+v", s.Counts)
	}
	if s.Counts["PLACED"] != 0 {
		t.Fatalf("PLACED 는 전이 후 0이어야 하는데 %d", s.Counts["PLACED"])
	}
	// 매출 = 배송 2건(1000+2000), 취소 1건은 빠짐.
	if s.TotalRevenue != 3000 {
		t.Fatalf("매출 = 3000 여야 하는데(취소 제외) %d", s.TotalRevenue)
	}
}

func TestProjector_상태로_검색(t *testing.T) {
	p, store := newProjector()
	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 1000))
	_ = p.Handle(statusEnv(t, "order.shipped", "order-1"))
	_ = p.Handle(placedEnv(t, "order-2", "cust-1", 2000)) // PLACED 로 남음

	shipped := store.Query("", "SHIPPED")
	if len(shipped) != 1 || shipped[0].OrderID != "order-1" {
		t.Fatalf("SHIPPED 검색은 order-1 1건이어야 하는데 %+v", shipped)
	}
	placed := store.Query("cust-1", "PLACED")
	if len(placed) != 1 || placed[0].OrderID != "order-2" {
		t.Fatalf("cust-1+PLACED 검색은 order-2 1건이어야 하는데 %+v", placed)
	}
}

func TestProjector_placed_재전송돼도_진행상태를_되돌리지_않는다(t *testing.T) {
	p, store := newProjector()
	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 5000))
	_ = p.Handle(statusEnv(t, "order.confirmed", "order-1"))
	// order.placed 가 다시 전달됨(중복)
	_ = p.Handle(placedEnv(t, "order-1", "cust-1", 5000))

	v, _ := store.Get("order-1")
	if v.Status != "CONFIRMED" {
		t.Fatalf("재전송에도 상태는 CONFIRMED 유지돼야 하는데 %s", v.Status)
	}
}
