package domain

import (
	"context"
	"errors"
)

var (
	ErrNonPositiveQuantity = errors.New("수량은 1 이상이어야 합니다")
	ErrCartNotFound        = errors.New("장바구니가 비어 있거나 없습니다")
	ErrEmptyCart           = errors.New("빈 장바구니로는 주문할 수 없습니다")
)

// CartRepository 는 장바구니 저장소 포트.
type CartRepository interface {
	Save(ctx context.Context, cart *Cart) error
	Find(ctx context.Context, customerID string) (*Cart, error)
	Delete(ctx context.Context, customerID string) error
}
