// Package readmodel 은 CQRS 의 "읽기 쪽"이다.
// 쓰기 모델(주문 서비스)이 낸 이벤트를 모아, 조회에 최적화된 비정규화 뷰를 만든다.
// 같은 이벤트 스트림에서 서로 다른 질의에 특화된 여러 뷰를 뽑을 수 있다 —
// 여기서는 (1) 주문별/회원별 조회, (2) 상태 필터, (3) 상태별 건수·매출 집계.
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

// Stats 는 집계 뷰 — 상태별 건수와 매출.
type Stats struct {
	Counts       map[string]int `json:"counts"`        // 상태별 주문 수
	OrderCount   int            `json:"order_count"`   // 전체 주문 수
	TotalRevenue int64          `json:"total_revenue"` // 취소를 뺀 총 주문액
}

// Store 는 읽기 모델 저장소 포트.
type Store interface {
	Upsert(v OrderView)
	SetStatus(orderID, status string)
	Get(orderID string) (OrderView, bool)
	ByCustomer(customerID string) []OrderView
	Query(customerID, status string) []OrderView // 둘 다 선택(빈 문자열이면 무시)
	Stats() Stats
}

// MemoryStore 는 읽기 모델을 메모리로 구현한 어댑터.
// 상태별 건수·매출은 이벤트가 흐를 때마다 증분으로 갱신한다(집계도 하나의 프로젝션).
type MemoryStore struct {
	mu      sync.RWMutex
	orders  map[string]OrderView
	counts  map[string]int
	revenue int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{orders: make(map[string]OrderView), counts: make(map[string]int)}
}

// Upsert 는 주문 뷰를 넣는다. 이미 있으면 진행된 상태는 되돌리지 않는다(재전송 방어).
func (s *MemoryStore) Upsert(v OrderView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.orders[v.OrderID]; ok {
		return // 이미 아는 주문 — 재전송이므로 집계를 두 번 세지 않는다(멱등)
	}
	s.orders[v.OrderID] = v
	s.counts[v.Status]++
	s.revenue += v.Total
}

func (s *MemoryStore) SetStatus(orderID, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.orders[orderID]
	if !ok || v.Status == status {
		return
	}
	s.counts[v.Status]--
	s.counts[status]++
	if status == "CANCELLED" || status == "REFUNDED" {
		s.revenue -= v.Total // 취소·환불된 주문은 매출에서 뺀다
	}
	v.Status = status
	s.orders[orderID] = v
}

func (s *MemoryStore) Get(orderID string) (OrderView, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.orders[orderID]
	return v, ok
}

func (s *MemoryStore) ByCustomer(customerID string) []OrderView {
	return s.Query(customerID, "")
}

// Query 는 회원·상태로 거른 주문을 주문 ID 순으로 돌려준다(둘 다 선택적).
func (s *MemoryStore) Query(customerID, status string) []OrderView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []OrderView
	for _, v := range s.orders {
		if customerID != "" && v.CustomerID != customerID {
			continue
		}
		if status != "" && v.Status != status {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OrderID < out[j].OrderID })
	return out
}

// Stats 는 상태별 건수·매출 집계를 돌려준다(증분으로 유지돼 훑지 않고 즉답).
func (s *MemoryStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := make(map[string]int, len(s.counts))
	for k, v := range s.counts {
		if v != 0 {
			counts[k] = v
		}
	}
	return Stats{Counts: counts, OrderCount: len(s.orders), TotalRevenue: s.revenue}
}
