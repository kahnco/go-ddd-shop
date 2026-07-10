package domain

import "context"

// StockRepository 는 재고 저장소 포트. 구현은 인프라 계층이 채운다(인메모리 → 뒤에서 DB).
type StockRepository interface {
	FindByProduct(ctx context.Context, id ProductID) (*StockItem, error)
	Save(ctx context.Context, item *StockItem) error
}

// ReservationRepository 는 주문별 예약 기록 저장소 포트. 취소 시 복원에 쓰인다.
type ReservationRepository interface {
	Save(ctx context.Context, reservation *Reservation) error
	Find(ctx context.Context, orderID OrderID) (*Reservation, error)
	Delete(ctx context.Context, orderID OrderID) error
}
