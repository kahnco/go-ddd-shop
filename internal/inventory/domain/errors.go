package domain

import "errors"

var (
	ErrNonPositiveQuantity = errors.New("수량은 1 이상이어야 합니다")
	ErrInsufficientStock   = errors.New("재고가 부족합니다")
	ErrStockItemNotFound   = errors.New("해당 상품의 재고 정보를 찾을 수 없습니다")
	ErrReservationNotFound = errors.New("해당 주문의 예약 기록을 찾을 수 없습니다")
)
