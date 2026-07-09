package domain

import "context"

// PaymentRepository 는 결제 저장소 포트. 구현은 인프라 계층이 채운다.
type PaymentRepository interface {
	Save(ctx context.Context, payment *Payment) error
}
