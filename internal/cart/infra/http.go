package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/cart/app"
	"github.com/kahnco/go-ddd-shop/internal/cart/domain"
)

// CartHandler 는 장바구니의 HTTP 어댑터.
type CartHandler struct {
	svc *app.CartService
}

func NewCartHandler(svc *app.CartService) *CartHandler {
	return &CartHandler{svc: svc}
}

func (h *CartHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /carts/{customerId}", h.get)
	mux.HandleFunc("POST /carts/{customerId}/items", h.addItem)
	mux.HandleFunc("DELETE /carts/{customerId}/items/{productId}", h.removeItem)
	mux.HandleFunc("POST /carts/{customerId}/checkout", h.checkout)
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
	cart, err := h.svc.Get(r.Context(), r.PathValue("customerId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) addItem(w http.ResponseWriter, r *http.Request) {
	var req itemView
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}
	cart, err := h.svc.AddItem(r.Context(), r.PathValue("customerId"), req.ProductID, req.Quantity)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) removeItem(w http.ResponseWriter, r *http.Request) {
	cart, err := h.svc.RemoveItem(r.Context(), r.PathValue("customerId"), r.PathValue("productId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toCartView(cart))
}

func (h *CartHandler) checkout(w http.ResponseWriter, r *http.Request) {
	orderID, err := h.svc.Checkout(r.Context(), r.PathValue("customerId"))
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
