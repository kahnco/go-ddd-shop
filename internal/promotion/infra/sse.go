package infra

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
)

// Hub 는 (이벤트, 사용자)별 SSE 구독자에게 응모 결과를 밀어 준다.
// 큐 완충 모드에서 접수(202) 뒤, 순차 소비자가 순번을 배정하면 그 결과를 바로 push 한다.
// readmodel 의 Hub(23편)와 같은 패턴 — 논블로킹, 느린 구독자는 드롭.
type Hub struct {
	mu   sync.Mutex
	subs map[string]map[chan []byte]struct{}
}

func NewHub() *Hub { return &Hub{subs: make(map[string]map[chan []byte]struct{})} }

func hubKey(eventID, userID string) string { return eventID + "|" + userID }

func (h *Hub) Subscribe(key string) chan []byte {
	ch := make(chan []byte, 4)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[key] == nil {
		h.subs[key] = make(map[chan []byte]struct{})
	}
	h.subs[key][ch] = struct{}{}
	return ch
}

func (h *Hub) Unsubscribe(key string, ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.subs[key]; ok {
		delete(set, ch)
		if len(set) == 0 {
			delete(h.subs, key)
		}
	}
	close(ch)
}

func (h *Hub) Publish(key string, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[key] {
		select {
		case ch <- data:
		default: // 밀린 구독자는 이번 갱신을 건너뛴다(다음 갱신이 최신)
		}
	}
}

// NotifyEntry 는 한 사용자의 응모 결과를 그 사용자 스트림으로 밀어 준다.
func (h *Hub) NotifyEntry(eventID, userID string, res app.Result) {
	h.Publish(hubKey(eventID, userID), statusJSON(res, true))
}

func statusJSON(res app.Result, found bool) []byte {
	if !found {
		b, _ := json.Marshal(map[string]string{"status": "processing"})
		return b
	}
	b, _ := json.Marshal(entryResponse{Seq: res.Seq, Winner: res.Winner, Already: res.Already})
	return b
}

// SSEHandler 는 GET /events/{eventId}/entries/me/stream 으로 내 응모 결과를 실시간으로 흘린다.
// 연결 직후 현재 상태(스냅샷)를 한 번 보내고, 이후 배정 결과를 push 받는다.
type SSEHandler struct {
	svc *app.Service
	hub *Hub
}

func NewSSEHandler(svc *app.Service, hub *Hub) *SSEHandler { return &SSEHandler{svc: svc, hub: hub} }

func (h *SSEHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /events/{eventId}/entries/me/stream", h.stream)
}

func (h *SSEHandler) stream(w http.ResponseWriter, r *http.Request) {
	eventID := r.PathValue("eventId")
	userID := userOf(r)
	if userID == "" {
		http.Error(w, "user_required", http.StatusUnauthorized)
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

	ch := h.hub.Subscribe(hubKey(eventID, userID))
	defer h.hub.Unsubscribe(hubKey(eventID, userID), ch)

	// 연결 직후 현재 상태 스냅샷(이미 배정됐을 수도 있으니).
	res, found, err := h.svc.EntryOf(r.Context(), eventID, userID)
	if err == nil {
		sseSend(w, flusher, statusJSON(res, found))
	}

	ping := time.NewTicker(15 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case data := <-ch:
			sseSend(w, flusher, data)
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func sseSend(w http.ResponseWriter, f http.Flusher, data []byte) {
	fmt.Fprintf(w, "data: %s\n\n", data)
	f.Flush()
}
