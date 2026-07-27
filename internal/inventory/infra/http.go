package infra

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// StockHandler 는 재고 조회용 최소 HTTP 어댑터.
// 프런트엔드가 상품 페이지에서 재고/품절을 표시하려고 GET /stock/{productId} 로 묻는다.
// 비즈니스 입구는 여전히 이벤트지만, 조회(read)는 값싸게 HTTP 로 연다.
type StockHandler struct {
	stock domain.StockRepository
}

func NewStockHandler(stock domain.StockRepository) *StockHandler {
	return &StockHandler{stock: stock}
}

func (h *StockHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /stock/{productId}", h.get)
}

func (h *StockHandler) get(w http.ResponseWriter, r *http.Request) {
	id := domain.ProductID(r.PathValue("productId"))
	item, err := h.stock.FindByProduct(r.Context(), id)
	available := 0
	if err == nil {
		available = item.Available()
	} else if !errors.Is(err, domain.ErrStockItemNotFound) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	// 재고 항목이 없으면 available=0(품절)으로 답한다.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"product_id": string(id),
		"available":  available,
	})
}
