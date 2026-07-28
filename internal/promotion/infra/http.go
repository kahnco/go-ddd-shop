package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// EntryHandler 는 응모 HTTP 입구다. POST /events/{eventId}/entries
// 사용자 식별은 X-User-Id 헤더로 받는다(실서비스에선 세션/JWT 에서 온다 — 13편 참고).
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
	eventID := req.PathValue("eventId")
	userID := req.Header.Get("X-User-Id")
	if userID == "" {
		writeErr(w, http.StatusUnauthorized, "user_required")
		return
	}

	res, err := h.svc.Enter(req.Context(), eventID, userID)
	switch {
	case errors.Is(err, domain.ErrEventNotFound):
		writeErr(w, http.StatusNotFound, "event_not_found")
		return
	case errors.Is(err, domain.ErrNotStarted):
		writeErr(w, http.StatusTooEarly, "not_started") // 425 — 아직 시작 전
		return
	case err != nil:
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, entryResponse{Seq: res.Seq, Winner: res.Winner, Already: res.Already})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
