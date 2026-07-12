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

	shippingapp "github.com/kahnco/go-ddd-shop/internal/shipping/app"
	shippinginfra "github.com/kahnco/go-ddd-shop/internal/shipping/infra"

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
	refund := paymentinfra.NewReturnRequestedConsumer(svc, discardLogger())
	if err := bus.Subscribe("ordering.order.return_requested", "payment", refund.Handle); err != nil {
		t.Fatalf("payment 환불 구독: %v", err)
	}
}

// wireOrdering 은 주문 서비스 + 핵심 사가 구독을 구성하고, 검사용 저장소·서비스·사가를 돌려준다.
// (배송 구독은 테스트별로 선택적으로 붙인다 — 확정에서 멈추는 테스트와 배송까지 가는 테스트를 나누기 위해.)
func wireOrdering(t *testing.T, bus *eventbus.Bus) (*orderinginfra.MemoryOrderRepository, *orderingapp.OrderService, *orderinginfra.OrderSagaConsumer) {
	t.Helper()
	repo := orderinginfra.NewMemoryOrderRepository()
	pub := orderinginfra.NewNatsEventPublisher(bus, "ordering")
	prices := orderinginfra.NewProductProjection()
	prices.SeedDefault("prod-A", 1000)
	prices.SeedDefault("prod-B", 3000)
	prices.SeedDefault("prod-X", 2_000_000) // 결제 한도 초과 테스트용
	svc := orderingapp.NewOrderService(repo, pub, fixedOrderID{id: "order-1"}, prices)

	saga := orderinginfra.NewOrderSagaConsumer(svc, discardLogger())
	subs := map[string]eventbus.Handler{
		"payment.completed":                  saga.OnPaymentCompleted,
		"payment.failed":                     saga.OnPaymentFailed,
		"payment.refunded":                   saga.OnPaymentRefunded,
		"inventory.stock.reservation_failed": saga.OnStockReservationFailed,
	}
	for subject, handler := range subs {
		if err := bus.Subscribe(subject, "ordering", handler); err != nil {
			t.Fatalf("%s 구독: %v", subject, err)
		}
	}
	return repo, svc, saga
}

// wireShipping 은 배송 컨텍스트를 구성하고 order.confirmed 를 구독시킨다.
func wireShipping(t *testing.T, bus *eventbus.Bus) {
	t.Helper()
	repo := shippinginfra.NewMemoryShipmentRepository()
	pub := shippinginfra.NewNatsEventPublisher(bus)
	svc := shippingapp.NewShippingService(repo, pub)
	consumer := shippinginfra.NewOrderConfirmedConsumer(svc, discardLogger())
	if err := bus.Subscribe("ordering.order.confirmed", "shipping", consumer.Handle); err != nil {
		t.Fatalf("shipping 구독: %v", err)
	}
}

func placeSampleOrder(t *testing.T, svc *orderingapp.OrderService) {
	t.Helper()
	_, err := svc.PlaceOrder(context.Background(), orderingapp.PlaceOrderCommand{
		CustomerID: "c1",
		Items: []orderingapp.OrderItemInput{
			{ProductID: "prod-A", Quantity: 2},
			{ProductID: "prod-B", Quantity: 1},
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
	orderRepo, orderSvc, _ := wireOrdering(t, bus)

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
	orderRepo, orderSvc, _ := wireOrdering(t, bus)

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

// 전체 흐름: 주문 → 재고 → 결제 → 확정 → 배송 이 이벤트만으로 SHIPPED 까지 가는지 본다.
func TestSaga_전체흐름이면_주문이_배송중까지_간다(t *testing.T) {
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

	wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 5})
	wirePayment(t, bus)
	orderRepo, orderSvc, saga := wireOrdering(t, bus)
	wireShipping(t, bus)
	// 배송 시작 → 주문 배송중 (이 테스트에서만 붙인다)
	if err := bus.Subscribe("shipping.dispatched", "ordering", saga.OnShipmentDispatched); err != nil {
		t.Fatal(err)
	}

	shipped := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.shipped", "test", func(e eventbus.Envelope) error {
		shipped <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	placeSampleOrder(t, orderSvc)

	select {
	case <-shipped:
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: 배송중(order.shipped)까지 흐르지 않음")
	}

	got, _ := orderRepo.FindByID(context.Background(), "order-1")
	if got.Status() != orderingdomain.StatusShipped {
		t.Fatalf("주문 상태 = SHIPPED 여야 하는데 %s", got.Status())
	}
}

// 반품: 배송된 주문의 반품을 요청하면 환불되고 재고가 재입고되며 주문이 REFUNDED 가 되는지 본다.
func TestSaga_반품요청이면_환불되고_재입고된다(t *testing.T) {
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

	stockRepo := wireInventory(t, bus, map[string]int{"prod-A": 10, "prod-B": 5})
	wirePayment(t, bus)
	orderRepo, orderSvc, saga := wireOrdering(t, bus)
	wireShipping(t, bus)
	if err := bus.Subscribe("shipping.dispatched", "ordering", saga.OnShipmentDispatched); err != nil {
		t.Fatal(err)
	}

	shipped := make(chan eventbus.Envelope, 1)
	_ = bus.Subscribe("ordering.order.shipped", "test", func(e eventbus.Envelope) error { shipped <- e; return nil })
	refunded := make(chan eventbus.Envelope, 1)
	_ = bus.Subscribe("ordering.order.refunded", "test", func(e eventbus.Envelope) error { refunded <- e; return nil })

	placeSampleOrder(t, orderSvc) // prod-A x2, prod-B x1 → 예약 후 A=8, B=4
	select {
	case <-shipped:
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: 배송까지 흐르지 않음")
	}

	// 배송된 주문에 반품 요청 → 환불 + 재입고 사가
	if err := orderSvc.RequestReturn(context.Background(), "order-1"); err != nil {
		t.Fatalf("RequestReturn: %v", err)
	}
	select {
	case <-refunded:
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: 환불(order.refunded)까지 흐르지 않음")
	}

	got, _ := orderRepo.FindByID(context.Background(), "order-1")
	if got.Status() != orderingdomain.StatusRefunded {
		t.Fatalf("주문 상태 = REFUNDED 여야 하는데 %s", got.Status())
	}
	// 반품 재입고로 재고가 원래대로(A=10, B=5) 돌아와야 한다.
	var restocked bool
	for i := 0; i < 50; i++ {
		a, _ := stockRepo.FindByProduct(context.Background(), invDomainProductID("prod-A"))
		b, _ := stockRepo.FindByProduct(context.Background(), invDomainProductID("prod-B"))
		if a.Available() == 10 && b.Available() == 5 {
			restocked = true
			break
		}
		<-time.After(50 * time.Millisecond)
	}
	if !restocked {
		a, _ := stockRepo.FindByProduct(context.Background(), invDomainProductID("prod-A"))
		t.Fatalf("반품 재입고 후 prod-A 재고 = 10 이어야 하는데 %d", a.Available())
	}
}

// 결제 실패: 한도를 넘는 주문은 결제가 거절되고, 주문 취소 + 예약 재고 복원까지 되는지 본다.
func TestSaga_결제실패면_취소되고_재고가_복원된다(t *testing.T) {
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

	stockRepo := wireInventory(t, bus, map[string]int{"prod-X": 10})
	wirePayment(t, bus)
	orderRepo, orderSvc, _ := wireOrdering(t, bus)

	cancelled := make(chan eventbus.Envelope, 1)
	if err := bus.Subscribe("ordering.order.cancelled", "test", func(e eventbus.Envelope) error {
		cancelled <- e
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// prod-X 가격 2,000,000원 → 결제 한도(1,000,000) 초과 → 거절.
	if _, err := orderSvc.PlaceOrder(context.Background(), orderingapp.PlaceOrderCommand{
		CustomerID: "c1",
		Items:      []orderingapp.OrderItemInput{{ProductID: "prod-X", Quantity: 1}},
	}); err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}

	select {
	case <-cancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("타임아웃: 결제 실패 → 주문 취소까지 흐르지 않음")
	}

	// 주문은 취소되고
	got, _ := orderRepo.FindByID(context.Background(), "order-1")
	if got.Status() != orderingdomain.StatusCancelled {
		t.Fatalf("주문 상태 = CANCELLED 여야 하는데 %s", got.Status())
	}
	// 잡아 뒀던 prod-A 1개는 복원돼 10으로 돌아와야 한다(완전한 보상).
	// (이벤트 전파가 비동기라 잠깐 기다린 뒤 확인)
	var restored bool
	for i := 0; i < 50; i++ {
		x, _ := stockRepo.FindByProduct(context.Background(), invDomainProductID("prod-X"))
		if x.Available() == 10 {
			restored = true
			break
		}
		<-time.After(50 * time.Millisecond)
	}
	if !restored {
		x, _ := stockRepo.FindByProduct(context.Background(), invDomainProductID("prod-X"))
		t.Fatalf("복원 후 prod-X 재고 = 10 이어야 하는데 %d", x.Available())
	}
}
