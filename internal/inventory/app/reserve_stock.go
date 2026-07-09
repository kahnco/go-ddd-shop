package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// ReserveForOrderCommand 는 "주문에 대해 재고를 예약하라"는 입력.
// 주문 컨텍스트의 OrderPlaced 이벤트에서 번역돼 들어온다.
type ReserveForOrderCommand struct {
	OrderID string
	Amount  int64 // 주문 총액. 예약 성공 시 결제 컨텍스트로 그대로 전달한다.
	Items   []ReservationItem
}

type ReservationItem struct {
	ProductID string
	Quantity  int
}

// ReservationService 는 재고 예약 유스케이스를 담는 애플리케이션 서비스.
type ReservationService struct {
	stock     domain.StockRepository
	publisher EventPublisher
}

func NewReservationService(stock domain.StockRepository, publisher EventPublisher) *ReservationService {
	return &ReservationService{stock: stock, publisher: publisher}
}

// OnOrderPlaced 는 주문 생성에 반응해 모든 항목의 재고를 예약한다.
// 전부 성공하면 StockReserved 를, 하나라도 실패하면 이미 잡은 예약을 되돌리고(보상)
// StockReservationFailed 를 발행한다 — "모두 아니면 전무(all-or-nothing)"를
// 분산 환경에서 흉내 내는, 사가(saga)의 축소판이다.
func (s *ReservationService) OnOrderPlaced(ctx context.Context, cmd ReserveForOrderCommand) error {
	type reserved struct {
		item *domain.StockItem
		qty  int
	}
	var done []reserved

	// 지금까지 잡은 예약을 전부 되돌린다(보상).
	rollback := func() {
		for _, r := range done {
			r.item.Release(r.qty)
			_ = s.stock.Save(ctx, r.item)
		}
	}

	for _, it := range cmd.Items {
		item, err := s.stock.FindByProduct(ctx, domain.ProductID(it.ProductID))
		if err != nil {
			rollback()
			return s.publishFailed(ctx, cmd.OrderID, err)
		}
		if err := item.Reserve(it.Quantity); err != nil {
			rollback()
			return s.publishFailed(ctx, cmd.OrderID, err)
		}
		if err := s.stock.Save(ctx, item); err != nil {
			rollback()
			return err // 인프라 오류는 그대로 올려보낸다(재시도 대상)
		}
		done = append(done, reserved{item: item, qty: it.Quantity})
	}

	return s.publisher.Publish(ctx, domain.StockReserved{OrderID: domain.OrderID(cmd.OrderID), Amount: cmd.Amount})
}

func (s *ReservationService) publishFailed(ctx context.Context, orderID string, cause error) error {
	return s.publisher.Publish(ctx, domain.StockReservationFailed{
		OrderID: domain.OrderID(orderID),
		Reason:  cause.Error(),
	})
}
