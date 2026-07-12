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
		productID string
		qty       int
	}
	var done []reserved

	// 지금까지 잡은 예약을 전부 되돌린다(보상). 각 복원도 원자적 Update 로.
	rollback := func() {
		for _, r := range done {
			_ = s.stock.Update(ctx, domain.ProductID(r.productID), func(si *domain.StockItem) error {
				si.Release(r.qty)
				return nil
			})
		}
	}

	for _, it := range cmd.Items {
		// 조회·차감·저장을 한 원자 단위로 — 동시 예약과 겹쳐도 재고가 유실되지 않는다.
		err := s.stock.Update(ctx, domain.ProductID(it.ProductID), func(si *domain.StockItem) error {
			return si.Reserve(it.Quantity)
		})
		if err != nil {
			rollback()
			if isReservationFailure(err) {
				return s.publishFailed(ctx, cmd.OrderID, err)
			}
			return err // 인프라 오류는 그대로 올려보낸다(재시도 대상)
		}
		done = append(done, reserved{productID: it.ProductID, qty: it.Quantity})
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
		_ = s.stock.Update(ctx, item.ProductID, func(si *domain.StockItem) error {
			si.Release(item.Quantity)
			return nil
		})
	}
	// 되돌렸으니 기록을 지운다 — 중복 취소가 와도 두 번 복원하지 않는다(멱등).
	return s.reservations.Delete(ctx, domain.OrderID(orderID))
}

// OnReturnRequested 는 반품 요청에 반응해 반품 상품을 재고로 다시 채운다.
// 반품된 항목은 이벤트에 실려 오므로, 예약 기록 없이도 처리할 수 있다.
func (s *ReservationService) OnReturnRequested(ctx context.Context, cmd ReserveForOrderCommand) error {
	for _, it := range cmd.Items {
		_ = s.stock.Update(ctx, domain.ProductID(it.ProductID), func(si *domain.StockItem) error {
			si.Restock(it.Quantity)
			return nil
		})
	}
	return s.publisher.Publish(ctx, domain.StockRestocked{OrderID: domain.OrderID(cmd.OrderID)})
}

func (s *ReservationService) publishFailed(ctx context.Context, orderID string, cause error) error {
	return s.publisher.Publish(ctx, domain.StockReservationFailed{
		OrderID: domain.OrderID(orderID),
		Reason:  cause.Error(),
	})
}

// isReservationFailure 는 "예약 실패"로 다뤄 보상+실패이벤트로 넘길 도메인 오류인지 판별한다.
func isReservationFailure(err error) bool {
	return errors.Is(err, domain.ErrStockItemNotFound) ||
		errors.Is(err, domain.ErrInsufficientStock) ||
		errors.Is(err, domain.ErrNonPositiveQuantity)
}
