package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// 주문 생명주기 유스케이스들. 이벤트(결제 완료·재고 부족 등)에 반응해
// 저장된 주문을 불러와 상태를 전이하고, 다시 저장하며, 새 이벤트를 발행한다.
// 규칙(어떤 전이가 허용되는가)은 전부 도메인(Order)이 판단한다.

// ConfirmPaidOrder 는 결제 완료에 반응해 주문을 결제완료→확정으로 전이한다.
// (재고는 이미 예약됐고 결제도 끝났으니, 주문을 확정한다.)
func (s *OrderService) ConfirmPaidOrder(ctx context.Context, id domain.OrderID) error {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := order.MarkPaid(); err != nil {
		return err
	}
	if err := order.Confirm(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, order.PullEvents()...)
}

// ShipOrder 는 배송 시작에 반응해 주문을 확정→배송중으로 전이한다.
func (s *OrderService) ShipOrder(ctx context.Context, id domain.OrderID) error {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := order.Ship(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, order.PullEvents()...)
}

// RequestReturn 은 고객의 반품 요청에 반응해 배송된 주문의 반품을 시작한다(사후 보상 사가).
func (s *OrderService) RequestReturn(ctx context.Context, id domain.OrderID) error {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := order.RequestReturn(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, order.PullEvents()...)
}

// MarkOrderRefunded 는 환불 완료(payment.refunded)에 반응해 주문을 환불완료로 전이한다.
func (s *OrderService) MarkOrderRefunded(ctx context.Context, id domain.OrderID) error {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := order.MarkRefunded(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, order.PullEvents()...)
}

// CancelOrder 는 재고 부족·결제 실패 등에 반응해 주문을 취소한다(사가의 보상 경로).
func (s *OrderService) CancelOrder(ctx context.Context, id domain.OrderID) error {
	order, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := order.Cancel(); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, order); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, order.PullEvents()...)
}
