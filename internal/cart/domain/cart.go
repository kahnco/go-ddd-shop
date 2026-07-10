// Package domain 은 장바구니(cart) bounded context 의 도메인이다.
package domain

import "sort"

// Cart 는 한 회원의 장바구니. 같은 상품은 수량을 합친다.
type Cart struct {
	customerID string
	items      map[string]int // productID -> quantity
}

func NewCart(customerID string) *Cart {
	return &Cart{customerID: customerID, items: make(map[string]int)}
}

// AddItem 은 상품을 담는다. 같은 상품이면 수량을 더한다.
func (c *Cart) AddItem(productID string, quantity int) error {
	if quantity <= 0 {
		return ErrNonPositiveQuantity
	}
	c.items[productID] += quantity
	return nil
}

// RemoveItem 은 상품을 장바구니에서 뺀다.
func (c *Cart) RemoveItem(productID string) {
	delete(c.items, productID)
}

func (c *Cart) CustomerID() string { return c.customerID }
func (c *Cart) IsEmpty() bool      { return len(c.items) == 0 }

// Items 는 담긴 항목을 상품 ID 순으로 돌려준다(결정적 순서).
func (c *Cart) Items() []CartItem {
	ids := make([]string, 0, len(c.items))
	for id := range c.items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]CartItem, 0, len(ids))
	for _, id := range ids {
		out = append(out, CartItem{ProductID: id, Quantity: c.items[id]})
	}
	return out
}

type CartItem struct {
	ProductID string
	Quantity  int
}

// Load 는 저장소가 기존 장바구니를 복원할 때 쓴다.
func Load(customerID string, items []CartItem) *Cart {
	c := NewCart(customerID)
	for _, it := range items {
		c.items[it.ProductID] = it.Quantity
	}
	return c
}
