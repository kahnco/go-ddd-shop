package domain

import "context"

// StockRepository 는 재고 저장소 포트.
//
// 재고 수정은 반드시 Update 로 한다 — 조회·수정·저장을 한 원자 단위로 묶어,
// 동시에 여러 요청이 같은 재고를 건드려도 read-modify-write 레이스가 나지 않게 한다.
// (Find→수정→Save 로 나누면 그 사이에 다른 요청이 끼어들어 갱신이 유실된다.)
type StockRepository interface {
	FindByProduct(ctx context.Context, id ProductID) (*StockItem, error) // 조회(스냅샷)
	Update(ctx context.Context, id ProductID, mutate func(*StockItem) error) error
}

// ReservationRepository 는 주문별 예약 기록 저장소 포트. 취소 시 복원에 쓰인다.
type ReservationRepository interface {
	Save(ctx context.Context, reservation *Reservation) error
	Find(ctx context.Context, orderID OrderID) (*Reservation, error)
	Delete(ctx context.Context, orderID OrderID) error
}
