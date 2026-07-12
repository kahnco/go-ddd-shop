package app

import (
	"context"
	"errors"

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
	stock        domain.StockRepository
	reservations domain.ReservationRepository
	publisher    EventPublisher
}

func NewReservationService(stock domain.StockRepository, reservations domain.ReservationRepository, publisher EventPublisher) *ReservationService {
	return &ReservationService{stock: stock, reservations: reservations, publisher: publisher}
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

	// 예약 성공 → 무엇을 예약했는지 기록해 둔다. 나중에 취소되면 이걸 보고 되돌린다.
	reservation := domain.NewReservation(domain.OrderID(cmd.OrderID))
	for _, it := range cmd.Items {
		reservation.Add(domain.ProductID(it.ProductID), it.Quantity)
	}
	if err := s.reservations.Save(ctx, reservation); err != nil {
		return err
	}

	return s.publisher.Publish(ctx, domain.StockReserved{OrderID: domain.OrderID(cmd.OrderID), Amount: cmd.Amount})
}

// OnOrderCancelled 는 주문 취소에 반응해, 그 주문을 위해 잡아 둔 재고를 되돌린다(보상).
// 예약 기록이 없으면(예: 애초에 예약이 실패한 주문) 할 일이 없으므로 조용히 넘어간다.
func (s *ReservationService) OnOrderCancelled(ctx context.Context, orderID string) error {
	reservation, err := s.reservations.Find(ctx, domain.OrderID(orderID))
	if errors.Is(err, domain.ErrReservationNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	for _, item := range reservation.Items() {
		stock, err := s.stock.FindByProduct(ctx, item.ProductID)
		if err != nil {
			continue // 재고 항목을 못 찾으면 건너뛴다(방어적)
		}
		stock.Release(item.Quantity)
		if err := s.stock.Save(ctx, stock); err != nil {
			return err
		}
	}
	// 되돌렸으니 기록을 지운다 — 중복 취소가 와도 두 번 복원하지 않는다(멱등).
	return s.reservations.Delete(ctx, domain.OrderID(orderID))
}

// OnReturnRequested 는 반품 요청에 반응해 반품 상품을 재고로 다시 채운다.
// 반품된 항목은 이벤트에 실려 오므로, 예약 기록 없이도 처리할 수 있다.
func (s *ReservationService) OnReturnRequested(ctx context.Context, cmd ReserveForOrderCommand) error {
	for _, it := range cmd.Items {
		stock, err := s.stock.FindByProduct(ctx, domain.ProductID(it.ProductID))
		if err != nil {
			continue // 모르는 상품이면 건너뛴다(방어적)
		}
		stock.Restock(it.Quantity)
		if err := s.stock.Save(ctx, stock); err != nil {
			return err
		}
	}
	return s.publisher.Publish(ctx, domain.StockRestocked{OrderID: domain.OrderID(cmd.OrderID)})
}

func (s *ReservationService) publishFailed(ctx context.Context, orderID string, cause error) error {
	return s.publisher.Publish(ctx, domain.StockReservationFailed{
		OrderID: domain.OrderID(orderID),
		Reason:  cause.Error(),
	})
}
