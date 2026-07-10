package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/catalog/app"
	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
)

// CatalogHandler 는 카탈로그의 HTTP 어댑터. 주문 서비스가 부트스트랩 때 GET /products 로 읽고,
// 운영자가 상품을 등록·가격 변경하는 입구다.
type CatalogHandler struct {
	svc *app.CatalogService
}

func NewCatalogHandler(svc *app.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /products", h.list)
	mux.HandleFunc("GET /products/{id}", h.get)
	mux.HandleFunc("POST /products", h.add)
	mux.HandleFunc("PUT /products/{id}/price", h.changePrice)
}

type productView struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	Price     int64  `json:"price"`
}

func toView(p *domain.Product) productView {
	return productView{ProductID: string(p.ID()), Name: p.Name(), Price: p.Price()}
}

func (h *CatalogHandler) list(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	views := make([]productView, 0, len(products))
	for _, p := range products {
		views = append(views, toView(p))
	}
	writeJSON(w, http.StatusOK, views)
}

func (h *CatalogHandler) get(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrProductNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, toView(p))
}

func (h *CatalogHandler) add(w http.ResponseWriter, r *http.Request) {
	var req productView
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}
	if err := h.svc.AddProduct(r.Context(), req.ProductID, req.Name, req.Price); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, req)
}

func (h *CatalogHandler) changePrice(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Price int64 `json:"price"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}
	err := h.svc.ChangePrice(r.Context(), r.PathValue("id"), req.Price)
	if errors.Is(err, domain.ErrProductNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"product_id": r.PathValue("id"), "price": req.Price})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
