// Package readmodel 은 CQRS 의 "읽기 쪽"이다.
// 쓰기 모델(주문 서비스)이 낸 이벤트를 모아, 조회에 최적화된 비정규화 뷰를 만든다.
// 예: "이 회원의 주문 목록" — 쓰기 모델은 주문 ID 로만 저장하니 이런 질의가 비싸지만,
// 읽기 모델은 회원별로 미리 모아 둬서 싸게 답한다.
package readmodel

import (
	"sort"
	"sync"
)

type Item struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// OrderView 는 조회용으로 비정규화한 주문 한 건.
type OrderView struct {
	OrderID    string `json:"order_id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
	Total      int64  `json:"total"`
	Items      []Item `json:"items"`
}

// Store 는 읽기 모델 저장소 포트.
type Store interface {
	Upsert(v OrderView)
	SetStatus(orderID, status string)
	Get(orderID string) (OrderView, bool)
	ByCustomer(customerID string) []OrderView
}

// MemoryStore 는 읽기 모델을 메모리로 구현한 어댑터.
// 읽기 모델은 이벤트로부터 언제든 다시 만들 수 있는 "파생 데이터"라, 유실돼도 재구축하면 된다.
type MemoryStore struct {
	mu     sync.RWMutex
	orders map[string]OrderView
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{orders: make(map[string]OrderView)}
}

// Upsert 는 주문 뷰를 넣는다. 이미 있으면 진행된 상태는 되돌리지 않는다
// (order.placed 가 재전송돼도 CONFIRMED 를 PLACED 로 깎지 않게).
func (s *MemoryStore) Upsert(v OrderView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.orders[v.OrderID]; ok {
		v.Status = existing.Status
	}
	s.orders[v.OrderID] = v
}

func (s *MemoryStore) SetStatus(orderID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.orders[orderID]; ok {
		v.Status = status
		s.orders[orderID] = v
	}
}

func (s *MemoryStore) Get(orderID string) (OrderView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.orders[orderID]
	return v, ok
}

// ByCustomer 는 한 회원의 주문을 주문 ID 순으로 돌려준다 — 읽기 모델의 존재 이유.
func (s *MemoryStore) ByCustomer(customerID string) []OrderView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []OrderView
	for _, v := range s.orders {
		if v.CustomerID == customerID {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OrderID < out[j].OrderID })
	return out
}
