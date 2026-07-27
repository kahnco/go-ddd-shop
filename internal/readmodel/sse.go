package readmodel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// ordersJSON 은 회원의 현재 주문 목록을 JSON 으로. SSE 초기 스냅샷·갱신 push 에 쓴다.
func ordersJSON(store Store, customerID string) []byte {
	orders := store.ByCustomer(customerID)
	if orders == nil {
		orders = []OrderView{}
	}
	data, _ := json.Marshal(orders)
	return data
}

// Hub 는 회원별 SSE 구독자에게 갱신을 밀어 준다.
// 프로젝터가 주문 이벤트를 반영할 때마다 그 회원의 채널로 최신 주문 목록을 보낸다.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{} // customerID → 구독 채널 집합
}

func NewHub() *Hub {
	return &Hub{subs: make(map[string]map[chan []byte]struct{})}
}

// Subscribe 는 회원의 갱신을 받을 채널을 만든다. 다 쓰면 Unsubscribe 로 정리한다.
func (h *Hub) Subscribe(customerID string) chan []byte {
	ch := make(chan []byte, 4) // 살짝 버퍼 — 느린 구독자가 프로젝터를 막지 않게
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[customerID] == nil {
		h.subs[customerID] = make(map[chan []byte]struct{})
	}
	h.subs[customerID][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(customerID string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.subs[customerID]; ok {
		delete(set, ch)
		if len(set) == 0 {
			delete(h.subs, customerID)
		}
	}
	close(ch)
}

// Publish 는 회원의 모든 구독자에게 데이터를 보낸다(논블로킹 — 버퍼가 차면 드롭).
func (h *Hub) Publish(customerID string, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[customerID] {
		select {
		case ch <- data:
		default: // 구독자가 밀렸으면 이 갱신은 건너뛴다(다음 갱신이 최신을 담는다)
		}
	}
}

// SSEHandler 는 GET /orders/stream?customer=X 로 실시간 주문 스트림을 연다.
type SSEHandler struct {
	store Store
	hub   *Hub
}

func NewSSEHandler(store Store, hub *Hub) *SSEHandler {
	return &SSEHandler{store: store, hub: hub}
}

func (h *SSEHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /orders/stream", h.stream)
}

func (h *SSEHandler) stream(w http.ResponseWriter, r *http.Request) {
	customerID := r.URL.Query().Get("customer")
	if customerID == "" {
		http.Error(w, "customer 쿼리 필요", http.StatusBadRequest)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "스트리밍 미지원", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := h.hub.Subscribe(customerID)
	defer h.hub.Unsubscribe(customerID, ch)

	// 연결 직후 현재 상태를 한 번 보낸다(초기 스냅샷).
	send(w, flusher, ordersJSON(h.store, customerID))

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done(): // 브라우저가 끊음
			return
		case data := <-ch:
			send(w, flusher, data)
		case <-ping.C:
			// 프록시가 유휴 연결을 끊지 않게 주석 라인으로 킵얼라이브.
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func send(w http.ResponseWriter, f http.Flusher, data []byte) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}
