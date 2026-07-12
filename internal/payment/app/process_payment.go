package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/payment/domain"
)

// EventPublisher 는 결제 컨텍스트가 자신의 이벤트를 발행하는 포트.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}

// ProcessPaymentCommand 는 "이 주문의 재고가 잡혔으니 결제하라"는 입력.
// 재고 컨텍스트의 StockReserved 이벤트에서 번역돼 들어온다.
type ProcessPaymentCommand struct {
	OrderID string
	Amount  int64
}

// PaymentService 는 결제 유스케이스를 담는 애플리케이션 서비스.
type PaymentService struct {
	repo      domain.PaymentRepository
	publisher EventPublisher
}

func NewPaymentService(repo domain.PaymentRepository, publisher EventPublisher) *PaymentService {
	return &PaymentService{repo: repo, publisher: publisher}
}

// OnStockReserved 는 재고 예약에 반응해 결제를 처리하고 결과 이벤트를 발행한다.
func (s *PaymentService) OnStockReserved(ctx context.Context, cmd ProcessPaymentCommand) error {
	payment := domain.NewPayment(domain.OrderID(cmd.OrderID), cmd.Amount)
	payment.Process() // 승인/거절 결정 + 이벤트 기록(도메인)

	if err := s.repo.Save(ctx, payment); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, payment.PullEvents()...)
}

// RefundCommand 는 "이 주문의 반품이 요청됐으니 환불하라"는 입력.
type RefundCommand struct {
	OrderID string
	Amount  int64
}

// OnReturnRequested 는 반품 요청에 반응해 환불을 처리하고 PaymentRefunded 를 발행한다.
// 데모용 목업 — 실제라면 PG 게이트웨이의 환불 API 를 호출한다.
func (s *PaymentService) OnReturnRequested(ctx context.Context, cmd RefundCommand) error {
	return s.publisher.Publish(ctx, domain.PaymentRefunded{
		OrderID: domain.OrderID(cmd.OrderID), Amount: cmd.Amount,
	})
}
