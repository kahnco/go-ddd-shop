package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/customer/app"
	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
)

// CustomerHandler 는 회원 컨텍스트의 HTTP 어댑터.
// 장바구니 서비스가 결제 시 GET /customers/{id} 로 회원 존재를 확인한다.
type CustomerHandler struct {
	svc *app.CustomerService
}

func NewCustomerHandler(svc *app.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /customers", h.register)
	mux.HandleFunc("GET /customers/{id}", h.get)
}

type customerView struct {
	CustomerID string `json:"customer_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
}

func (h *CustomerHandler) register(w http.ResponseWriter, r *http.Request) {
	var req customerView
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}
	err := h.svc.Register(r.Context(), req.CustomerID, req.Email, req.Name)
	switch {
	case errors.Is(err, domain.ErrCustomerExists):
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidCustomer):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusCreated, req)
	}
}

func (h *CustomerHandler) get(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrCustomerNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, customerView{CustomerID: string(c.ID()), Email: c.Email(), Name: c.Name()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
