package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// OrderHandler 는 HTTP 요청을 애플리케이션 유스케이스로 잇는 어댑터.
// 여기서는 JSON 디코딩/인코딩과 에러→상태코드 매핑만 한다. 규칙은 도메인에.
type OrderHandler struct {
	svc *app.OrderService
}

func NewOrderHandler(svc *app.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

// Register 는 Go 1.22+ ServeMux 의 "메서드 + 경로" 패턴으로 라우트를 건다.
func (h *OrderHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.placeOrder)
	mux.HandleFunc("GET /orders/{id}", h.getOrder)
	mux.HandleFunc("POST /orders/{id}/return", h.requestReturn) // 반품 요청(배송된 주문만)
}

// --- 요청/응답 DTO ---

type placeOrderRequest struct {
	CustomerID string `json:"customer_id"`
	Items      []struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
		// 가격(unit_price)은 받지 않는다 — 서버가 카탈로그에서 정한다.
	} `json:"items"`
}

type orderResponse struct {
	OrderID    string         `json:"order_id"`
	CustomerID string         `json:"customer_id"`
	Status     string         `json:"status"`
	Total      int64          `json:"total"`
	Items      []lineResponse `json:"items"`
}

type lineResponse struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// --- 핸들러 ---

func (h *OrderHandler) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req placeOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "잘못된 JSON"})
		return
	}

	cmd := app.PlaceOrderCommand{CustomerID: req.CustomerID}
	for _, it := range req.Items {
		cmd.Items = append(cmd.Items, app.OrderItemInput{
			ProductID: it.ProductID, Quantity: it.Quantity,
		})
	}

	id, err := h.svc.PlaceOrder(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"order_id": string(id)})
}

func (h *OrderHandler) requestReturn(w http.ResponseWriter, r *http.Request) {
	id := domain.OrderID(r.PathValue("id"))
	if err := h.svc.RequestReturn(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"order_id": string(id), "status": "RETURN_REQUESTED"})
}

func (h *OrderHandler) getOrder(w http.ResponseWriter, r *http.Request) {
	id := domain.OrderID(r.PathValue("id"))
	order, err := h.svc.GetOrder(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := orderResponse{
		OrderID:    string(order.ID()),
		CustomerID: string(order.CustomerID()),
		Status:     string(order.Status()),
		Total:      order.Total().Amount(),
	}
	for _, l := range order.Lines() {
		resp.Items = append(resp.Items, lineResponse{
			ProductID: string(l.ProductID()), Quantity: l.Quantity().Value(),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- 헬퍼: 도메인 에러를 HTTP 상태코드로 매핑 ---

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		status = http.StatusConflict // 예: 배송 안 된 주문에 반품 요청
	case errors.Is(err, domain.ErrEmptyOrder),
		errors.Is(err, domain.ErrNegativeMoney),
		errors.Is(err, domain.ErrNonPositiveQuantity),
		errors.Is(err, domain.ErrUnknownProduct):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
