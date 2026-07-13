package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/cart/app"
	"github.com/kahnco/go-ddd-shop/internal/cart/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/auth"
)

// CartHandler 는 장바구니의 HTTP 어댑터.
type CartHandler struct {
	svc *app.CartService
}

func NewCartHandler(svc *app.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

// Register 는 모든 장바구니 라우트를 인증 미들웨어로 감싼다.
// 경로의 {customerId} 는 토큰 신원과 일치해야 한다 — 남의 장바구니를 만질 수 없다.
func (h *CartHandler) Register(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.Handle("GET /carts/{customerId}", authMW(http.HandlerFunc(h.get)))
	mux.Handle("POST /carts/{customerId}/items", authMW(http.HandlerFunc(h.addItem)))
	mux.Handle("DELETE /carts/{customerId}/items/{productId}", authMW(http.HandlerFunc(h.removeItem)))
	mux.Handle("POST /carts/{customerId}/checkout", authMW(http.HandlerFunc(h.checkout)))
}

// ownCustomer 는 경로의 customerId 가 인증된 회원과 같은지 확인한다.
// 다르면 403 을 쓰고 false 를 돌려준다(핸들러는 즉시 반환해야 한다).
func ownCustomer(w http.ResponseWriter, r *http.Request) (string, bool) {
	pathID := r.PathValue("customerId")
	if pathID != auth.Subject(r.Context()) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "본인의 장바구니만 접근할 수 있습니다"})
		return "", false
	}
	return pathID, true
}

type cartView struct {
	CustomerID string     `json:"customer_id"`
	Items      []itemView `json:"items"`
}

type itemView struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

func toCartView(c *domain.Cart) cartView {
	v := cartView{CustomerID: c.CustomerID(), Items: []itemView{}}
	for _, it := range c.Items() {
		v.Items = append(v.Items, itemView{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	return v
}

func (h *CartHandler) get(w http.ResponseWriter, r *http.Request) {
	customerID, ok := ownCustomer(w, r)
	if !ok {
		return
	}
	cart, err := h.svc.Get(r.Context(), customerID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) addItem(w http.ResponseWriter, r *http.Request) {
	customerID, ok := ownCustomer(w, r)
	if !ok {
		return
	}
	var req itemView
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}
	cart, err := h.svc.AddItem(r.Context(), customerID, req.ProductID, req.Quantity)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) removeItem(w http.ResponseWriter, r *http.Request) {
	customerID, ok := ownCustomer(w, r)
	if !ok {
		return
	}
	cart, err := h.svc.RemoveItem(r.Context(), customerID, r.PathValue("productId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) checkout(w http.ResponseWriter, r *http.Request) {
	customerID, ok := ownCustomer(w, r)
	if !ok {
		return
	}
	orderID, err := h.svc.Checkout(r.Context(), customerID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"order_id": orderID})
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, app.ErrCustomerNotRegistered):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrEmptyCart), errors.Is(err, domain.ErrNonPositiveQuantity):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
