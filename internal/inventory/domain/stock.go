// Package domain 은 재고(inventory) bounded context 의 도메인이다.
// 주문 컨텍스트와 코드를 공유하지 않는다 — 오직 이벤트 계약(JSON)으로만 연결된다.
// 그래서 여기에도 자기만의 ProductID·OrderID 가 따로 있다.
package domain

// 재고 컨텍스트의 식별자. 주문 컨텍스트의 같은 이름 타입과는 별개다.
type (
	ProductID string
	OrderID   string
)

// StockItem 은 한 상품의 재고. 애그리거트 루트로서
// "가진 것보다 많이 예약할 수 없다"는 불변식을 스스로 지킨다.
type StockItem struct {
	productID ProductID
	available int
}

func NewStockItem(productID ProductID, available int) *StockItem {
	return &StockItem{productID: productID, available: available}
}

func (s *StockItem) ProductID() ProductID { return s.productID }
func (s *StockItem) Available() int       { return s.available }

// Reserve 는 재고를 예약(차감)한다. 수량이 0 이하이거나 재고보다 많으면 거부한다.
func (s *StockItem) Reserve(qty int) error {
	if qty <= 0 {
		return ErrNonPositiveQuantity
	}
	if qty > s.available {
		return ErrInsufficientStock
	}
	s.available -= qty
	return nil
}

// Release 는 예약을 되돌린다(차감분 복원). 보상 트랜잭션에서 쓰인다.
func (s *StockItem) Release(qty int) {
	if qty > 0 {
		s.available += qty
	}
}

// Restock 은 반품된 상품을 재고로 다시 채운다(사후 반품 처리).
func (s *StockItem) Restock(qty int) {
	if qty > 0 {
		s.available += qty
	}
}
