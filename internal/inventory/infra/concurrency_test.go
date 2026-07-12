package infra_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/inventory/app"
	"github.com/kahnco/go-ddd-shop/internal/inventory/infra"
)

// 원자적 Update 로 바꾼 뒤: 같은 상품에 1000개 예약이 동시에 몰려도
// 정확히 1000개가 예약돼 재고가 0이 되어야 한다(레이스·유실 없음).
// go test -race 로 돌려도 데이터 레이스가 잡히지 않아야 한다.
func TestConcurrency_동시_예약이_몰려도_재고가_정확하다(t *testing.T) {
	stock := infra.NewMemoryStockRepository()
	stock.Seed("prod-A", 1000)
	reservations := infra.NewMemoryReservationRepository()
	svc := app.NewReservationService(stock, reservations, nopPublisher{})

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = svc.OnOrderPlaced(context.Background(), app.ReserveForOrderCommand{
				OrderID: fmt.Sprintf("o-%d", n),
				Items:   []app.ReservationItem{{ProductID: "prod-A", Quantity: 1}},
			})
		}(i)
	}
	wg.Wait()

	a, _ := stock.FindByProduct(context.Background(), "prod-A")
	if a.Available() != 0 {
		t.Fatalf("동시 예약 1000건 후 재고 = 0 이어야 하는데 %d (유실 발생)", a.Available())
	}
}

// 예약과 반품(다른 구독 goroutine)이 같은 상품에 동시에 와도 재고가 어긋나지 않는지 본다.
func TestConcurrency_예약과_반품이_동시에_와도_어긋나지_않는다(t *testing.T) {
	stock := infra.NewMemoryStockRepository()
	stock.Seed("prod-A", 500)
	reservations := infra.NewMemoryReservationRepository()
	svc := app.NewReservationService(stock, reservations, nopPublisher{})

	var wg sync.WaitGroup
	// 500건 예약(재고 -500) 과 500건 반품 재입고(재고 +500) 를 동시에.
	for i := 0; i < 500; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = svc.OnOrderPlaced(context.Background(), app.ReserveForOrderCommand{
				OrderID: fmt.Sprintf("res-%d", n),
				Items:   []app.ReservationItem{{ProductID: "prod-A", Quantity: 1}},
			})
		}(i)
		go func(n int) {
			defer wg.Done()
			_ = svc.OnReturnRequested(context.Background(), app.ReserveForOrderCommand{
				OrderID: fmt.Sprintf("ret-%d", n),
				Items:   []app.ReservationItem{{ProductID: "prod-A", Quantity: 1}},
			})
		}(i)
	}
	wg.Wait()

	// -500 + 500 = 0 순변동. 최종 재고는 500 그대로여야 한다.
	a, _ := stock.FindByProduct(context.Background(), "prod-A")
	if a.Available() != 500 {
		t.Fatalf("예약·반품 상쇄 후 재고 = 500 이어야 하는데 %d", a.Available())
	}
}
