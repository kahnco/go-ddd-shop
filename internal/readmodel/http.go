package readmodel

import (
	"encoding/json"
	"net/http"
)

// QueryHandler 는 읽기 모델의 조회 HTTP API.
// 쓰기 쪽(주문 서비스)과 별개다 — 읽기는 여기, 쓰기는 저기(CQRS).
type QueryHandler struct {
	store Store
}

func NewQueryHandler(store Store) *QueryHandler {
	return &QueryHandler{store: store}
}

func (h *QueryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("GET /customers/{customerId}/orders", h.listByCustomer)
}

func (h *QueryHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	v, ok := h.store.Get(r.PathValue("id"))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "주문 뷰를 찾을 수 없습니다"})
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// listByCustomer 는 "내 주문 목록" — 읽기 모델이 있어서 싸게 답하는 질의.
func (h *QueryHandler) listByCustomer(w http.ResponseWriter, r *http.Request) {
	orders := h.store.ByCustomer(r.PathValue("customerId"))
	if orders == nil {
		orders = []OrderView{}
	}
	writeJSON(w, http.StatusOK, orders)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
