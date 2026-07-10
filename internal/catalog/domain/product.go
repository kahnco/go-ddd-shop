// Package domain 은 상품 카탈로그(catalog) bounded context 의 도메인이다.
// 상품과 가격의 소유자(source of truth)다 — 주문은 여기서 가격을 받아 온다.
package domain

type ProductID string

// Product 는 카탈로그의 상품. 이름과 가격(원)을 가진다.
type Product struct {
	id     ProductID
	name   string
	price  int64
	events []DomainEvent
}

// NewProduct 는 새 상품을 등록한다. ProductAdded 이벤트를 기록한다.
func NewProduct(id ProductID, name string, price int64) *Product {
	p := &Product{id: id, name: name, price: price}
	p.record(ProductAdded{ProductID: id, Name: name, Price: price})
	return p
}

// ChangePrice 는 가격을 바꾼다. ProductPriceChanged 이벤트를 기록한다.
// 이 이벤트가 주문 컨텍스트의 가격 프로젝션을 갱신한다.
func (p *Product) ChangePrice(newPrice int64) {
	p.price = newPrice
	p.record(ProductPriceChanged{ProductID: p.id, Price: newPrice})
}

func (p *Product) ID() ProductID        { return p.id }
func (p *Product) Name() string         { return p.name }
func (p *Product) Price() int64         { return p.price }
func (p *Product) record(e DomainEvent) { p.events = append(p.events, e) }
func (p *Product) PullEvents() []DomainEvent {
	e := p.events
	p.events = nil
	return e
}
