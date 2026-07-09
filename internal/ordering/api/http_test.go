package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
)

// 실제 인프라 어댑터(인메모리 저장소·로그 발행)를 그대로 끼운 통합 성격의 테스트.
// HTTP 요청 → 유스케이스 → 도메인 → 저장까지가 한 번에 도는지 본다.
func newTestServer() http.Handler {
	repo := infra.NewMemoryOrderRepository()
	pub := infra.NewLogPublisher(slog.New(slog.NewTextHandler(nopWriter{}, nil)))
	svc := app.NewOrderService(repo, pub, infra.RandomIDGenerator{})

	mux := http.NewServeMux()
	NewOrderHandler(svc).Register(mux)
	return mux
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestPlaceOrder_201_그리고_조회하면_같은_주문(t *testing.T) {
	srv := newTestServer()

	body := `{"customer_id":"cust-1","items":[
		{"product_id":"prod-A","quantity":2,"unit_price":1000},
		{"product_id":"prod-B","quantity":1,"unit_price":3000}]}`

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(body))
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("상태코드 = 201 여야 하는데 %d (%s)", rec.Code, rec.Body.String())
	}
	var created struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("응답 디코딩: %v", err)
	}
	if created.OrderID == "" {
		t.Fatal("order_id 가 비어 있음")
	}

	// 방금 만든 주문을 조회하면 총액 5000, 상태 PLACED 여야 한다.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/orders/"+created.OrderID, nil)
	srv.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("조회 상태코드 = 200 여야 하는데 %d", rec2.Code)
	}
	var view orderResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &view); err != nil {
		t.Fatalf("조회 응답 디코딩: %v", err)
	}
	if view.Total != 5000 {
		t.Fatalf("총액 = 5000 여야 하는데 %d", view.Total)
	}
	if view.Status != "PLACED" {
		t.Fatalf("상태 = PLACED 여야 하는데 %s", view.Status)
	}
	if len(view.Items) != 2 {
		t.Fatalf("항목 2개여야 하는데 %d개", len(view.Items))
	}
}

func TestPlaceOrder_항목이_없으면_400(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders",
		bytes.NewBufferString(`{"customer_id":"cust-1","items":[]}`))
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("빈 주문은 400 여야 하는데 %d", rec.Code)
	}
}

func TestGetOrder_없는_주문이면_404(t *testing.T) {
	srv := newTestServer()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/orders/order_none", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("없는 주문은 404 여야 하는데 %d", rec.Code)
	}
}
