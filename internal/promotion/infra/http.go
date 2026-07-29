package infra

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// 사용자 식별은 X-User-Id 헤더로 받는다(실서비스에선 세션/JWT 에서 온다 — 13편 참고).
func userOf(req *http.Request) string { return req.Header.Get("X-User-Id") }

// EntryHandler(동기 모드): POST /events/{eventId}/entries → 그 자리에서 순번을 배정해 응답한다.
type EntryHandler struct {
	svc *app.Service
}

func NewEntryHandler(svc *app.Service) *EntryHandler { return &EntryHandler{svc: svc} }

func (h *EntryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /events/{eventId}/entries", h.enter)
}

type entryResponse struct {
	Seq     int  `json:"seq"`
	Winner  bool `json:"winner"`
	Already bool `json:"already"`
}

func (h *EntryHandler) enter(w http.ResponseWriter, req *http.Request) {
	userID := userOf(req)
	if userID == "" {
		writeErr(w, http.StatusUnauthorized, "user_required")
		return
	}
	res, err := h.svc.Enter(req.Context(), req.PathValue("eventId"), userID)
	if handleEnterErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, entryResponse{Seq: res.Seq, Winner: res.Winner, Already: res.Already})
}

// QueuedEntryHandler(큐 완충 모드): POST /events/{eventId}/entries →
// 요청을 큐에 흘리고 즉시 202 를 돌려준다. 순번은 뒤에서 순차 소비자가 배정한다.
// 사용자는 상태 엔드포인트(GET …/entries/me)로 결과를 확인한다.
type QueuedEntryHandler struct {
	queue app.EntryQueue
}

func NewQueuedEntryHandler(queue app.EntryQueue) *QueuedEntryHandler {
	return &QueuedEntryHandler{queue: queue}
}

func (h *QueuedEntryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /events/{eventId}/entries", h.enqueue)
}

type queuedResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

func (h *QueuedEntryHandler) enqueue(w http.ResponseWriter, req *http.Request) {
	userID := userOf(req)
	if userID == "" {
		writeErr(w, http.StatusUnauthorized, "user_required")
		return
	}
	// 클라이언트가 Idempotency-Key 를 주면 그걸, 없으면 새로 만든다(재전송 시 중복 방지의 열쇠).
	requestID := req.Header.Get("Idempotency-Key")
	if requestID == "" {
		requestID = newRequestID()
	}
	if err := h.queue.Enqueue(req.Context(), req.PathValue("eventId"), userID, requestID); err != nil {
		writeErr(w, http.StatusInternalServerError, "enqueue_failed")
		return
	}
	writeJSON(w, http.StatusAccepted, queuedResponse{RequestID: requestID, Status: "queued"})
}

// StatusHandler: GET /events/{eventId}/entries/me → 내 응모 상태(순번·당첨).
// 아직 처리 전이면 202(processing).
type StatusHandler struct {
	svc *app.Service
}

func NewStatusHandler(svc *app.Service) *StatusHandler { return &StatusHandler{svc: svc} }

func (h *StatusHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /events/{eventId}/entries/me", h.status)
}

func (h *StatusHandler) status(w http.ResponseWriter, req *http.Request) {
	userID := userOf(req)
	if userID == "" {
		writeErr(w, http.StatusUnauthorized, "user_required")
		return
	}
	res, found, err := h.svc.EntryOf(req.Context(), req.PathValue("eventId"), userID)
	switch {
	case errors.Is(err, domain.ErrEventNotFound):
		writeErr(w, http.StatusNotFound, "event_not_found")
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if !found {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "processing"})
		return
	}
	writeJSON(w, http.StatusOK, entryResponse{Seq: res.Seq, Winner: res.Winner, Already: res.Already})
}

// handleEnterErr 는 Enter 오류를 HTTP 상태로 옮긴다. 처리했으면 true.
func handleEnterErr(w http.ResponseWriter, err error) bool {
	switch {
	case errors.Is(err, domain.ErrEventNotFound):
		writeErr(w, http.StatusNotFound, "event_not_found")
	case errors.Is(err, domain.ErrNotStarted):
		writeErr(w, http.StatusTooEarly, "not_started") // 425 — 아직 시작 전
	case errors.Is(err, domain.ErrClosed):
		writeErr(w, http.StatusConflict, "closed") // 409 — 이미 종료
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "internal")
	default:
		return false
	}
	return true
}

func newRequestID() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
