package domain

// 카탈로그가 발행하는 도메인 이벤트. 주문 컨텍스트가 이를 구독해 로컬 가격 프로젝션을 만든다.

type DomainEvent interface {
	EventName() string
}

// ProductAdded — 새 상품 등록됨.
type ProductAdded struct {
	ProductID ProductID `json:"product_id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
}

func (ProductAdded) EventName() string { return "product.added" }

// ProductPriceChanged — 상품 가격 변경됨. 주문의 가격 프로젝션이 이걸로 갱신된다.
type ProductPriceChanged struct {
	ProductID ProductID `json:"product_id"`
	Price     int64     `json:"price"`
}

func (ProductPriceChanged) EventName() string { return "product.price_changed" }
