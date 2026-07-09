package integration

import (
	"context"
	"testing"
	"time"

	orderingapp "github.com/kahnco/go-ddd-shop/internal/ordering/app"
	orderingdomain "github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	orderinginfra "github.com/kahnco/go-ddd-shop/internal/ordering/infra"

	paymentapp "github.com/kahnco/go-ddd-shop/internal/payment/app"
	paymentinfra "github.com/kahnco/go-ddd-shop/internal/payment/infra"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus/embeddednats"
)

// 고정 주문 ID 로 결과를 결정적으로 검증.
type fixedOrderID struct{ id orderingdomain.OrderID }

func (f fixedOrderID) NewOrderID() orderingdomain.OrderID { return f.id }

// wirePayment 는 결제 컨텍스트를 구성하고 stock.reserved 를 구독시킨다.
func wirePayment(t *testing.T, bus *eventbus.Bus) {
	t.Helper()
	repo := paymentinfra.NewMemoryPaymentRepository()
	pub := paymentinfra.NewNatsEventPublisher(bus)
	svc := paymentapp.NewPaymentService(repo, pub)
	consumer := paymentinfra.NewStockReservedConsumer(svc, discardLogger())
	if err := bus.Subscribe("inventory.stock.reserved", "payment", consumer.Handle); err != nil {
		t.Fatalf("payment 구독: %v", err)
	}
}

// wireOrdering 은 주문 서비스 + 사가 구독을 구성하고, 검사용 저장소와 서비스를 돌려준다.
func wireOrdering(t *testing.T, bus *eventbus.Bus) (*orderinginfra.MemoryOrderRepository, *orderingapp.OrderService) {
	t.Helper()
	repo := orderinginfra.NewMemoryOrderRepository()
	pub := orderinginfra.NewNatsEventPublisher(bus, "ordering")
	svc := orderingapp.NewOrderService(repo, pub, fixedOrderID{id: "order-1"})

	saga := orderinginfra.NewOrderSagaConsumer(svc, discardLogger())
	if err := bus.Subscribe("payment.completed", "ordering", saga.OnPaymentCompleted); err != nil {
		t.Fatalf("payment.completed 구독: %v", err)
	}
	if err := bus.Subscribe("inventory.stock.reservation_failed", "ordering", saga.OnStockReservationFailed); err != nil {
		t.Fatalf("stock.reservation_failed 구독: %v", err)
	}
	return repo, svc
}

func placeSampleOrder(t *testing.T, svc *orderingapp.OrderService) {
	t.Helper()
	_, err := svc.PlaceOrder(context.Background(), orderingapp.PlaceOrderCommand{
		CustomerID: "c1",
		Items: []orderingapp.OrderItemInput{
			{ProductID: "prod-A", Quantity: 2, UnitPrice: 1000},
			{ProductID: "prod-B", Quantity: 1, UnitPrice: 3000},
		},
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
}

// 해피 패스: 주문 → 재고예약 → 결제 → 확정 이 이벤트만으로 끝까지 흐르는지 본다.
func TestSaga_해피패스면_주문이_확정된다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()
	bus, err := eventbus.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 5}) // 재고 충분
	wirePayment(t, bus)
	orderRepo, orderSvc := wireOrdering(t, bus)

	confirmed := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.confirmed", "test", func(e eventbus.Envelope) error {
		confirmed <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	placeSampleOrder(t, orderSvc)

	select {
	case <-confirmed:
	case <-time.After(4 * time.Second):
		t.Fatal("타임아웃: 주문 확정(order.confirmed)까지 흐르지 않음")
	}

	got, _ := orderRepo.FindByID(context.Background(), "order-1")
	if got.Status() != orderingdomain.StatusConfirmed {
		t.Fatalf("주문 상태 = CONFIRMED 여야 하는데 %s", got.Status())
	}
}

// 실패 경로: 재고가 부족하면 stock.reservation_failed 로 주문이 자동 취소되는지 본다(보상).
func TestSaga_재고부족이면_주문이_자동취소된다(t *testing.T) {
	url, shutdown, err := embeddednats.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer shutdown()
	bus, err := eventbus.Connect(url)
	if err != nil {
		t.Fatal(err)
	}
	defer bus.Close()

	wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 0}) // prod-B 부족
	wirePayment(t, bus)
	orderRepo, orderSvc := wireOrdering(t, bus)

	cancelled := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.cancelled", "test", func(e eventbus.Envelope) error {
		cancelled <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	placeSampleOrder(t, orderSvc)

	select {
	case <-cancelled:
	case <-time.After(4 * time.Second):
		t.Fatal("타임아웃: 주문 취소(order.cancelled)까지 흐르지 않음")
	}

	got, _ := orderRepo.FindByID(context.Background(), "order-1")
	if got.Status() != orderingdomain.StatusCancelled {
		t.Fatalf("주문 상태 = CANCELLED 여야 하는데 %s", got.Status())
	}
}
