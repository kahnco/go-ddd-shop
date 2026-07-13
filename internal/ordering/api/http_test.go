package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/auth"
)

const testSecret = "ordering-test-secret"

// 실제 인프라 어댑터(인메모리 저장소·로그 발행)를 그대로 끼운 통합 성격의 테스트.
// HTTP 요청 → 인증 미들웨어 → 유스케이스 → 도메인 → 저장까지가 한 번에 도는지 본다.
func newTestServer() http.Handler {
	repo := infra.NewMemoryOrderRepository()
	pub := infra.NewLogPublisher(slog.New(slog.NewTextHandler(nopWriter{}, nil)))
	// 가격 프로젝션에 데모 상품을 시드(카탈로그 대신).
	prices := infra.NewProductProjection()
	prices.SeedDefault("prod-A", 1000)
	prices.SeedDefault("prod-B", 3000)
	svc := app.NewOrderService(repo, pub, infra.RandomIDGenerator{}, prices)

	mux := http.NewServeMux()
	NewOrderHandler(svc).Register(mux, auth.Middleware(testSecret))
	return mux
}

// authReq 는 주어진 회원으로 로그인한 것처럼 Authorization 헤더를 붙인 요청을 만든다.
func authReq(t *testing.T, customerID, method, path, body string) *http.Request {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
	}
	token, err := auth.Issue(testSecret, customerID, time.Hour, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestPlaceOrder_201_그리고_조회하면_같은_주문(t *testing.T) {
	srv := newTestServer()

	// 가격은 보내지 않는다 — 서버가 카탈로그에서 정한다(prod-A=1000, prod-B=3000 → 5000).
	// customer_id 도 보내지 않는다 — 신원은 토큰에서.
	body := `{"items":[
		{"product_id":"prod-A","quantity":2},
		{"product_id":"prod-B","quantity":1}]}`

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, authReq(t, "cust-1", http.MethodPost, "/orders", body))

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

	// 방금 만든 주문을 (같은 회원으로) 조회하면 총액 5000, 상태 PLACED 여야 한다.
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, authReq(t, "cust-1", http.MethodGet, "/orders/"+created.OrderID, ""))

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
	if view.CustomerID != "cust-1" {
		t.Fatalf("주문 주인 = cust-1 여야 하는데 %s", view.CustomerID)
	}
	if len(view.Items) != 2 {
		t.Fatalf("항목 2개여야 하는데 %d개", len(view.Items))
	}
}

func TestPlaceOrder_토큰없으면_401(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBufferString(`{"items":[]}`))
	srv.ServeHTTP(rec, req) // Authorization 없음
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("토큰 없으면 401 여야 하는데 %d", rec.Code)
	}
}

func TestGetOrder_남의_주문은_403(t *testing.T) {
	srv := newTestServer()

	// cust-1 이 주문을 만든다.
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, authReq(t, "cust-1", http.MethodPost, "/orders",
		`{"items":[{"product_id":"prod-A","quantity":1}]}`))
	var created struct {
		OrderID string `json:"order_id"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)

	// cust-2 가 그 주문을 조회하면 403.
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, authReq(t, "cust-2", http.MethodGet, "/orders/"+created.OrderID, ""))
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("남의 주문 조회는 403 여야 하는데 %d", rec2.Code)
	}
}

func TestPlaceOrder_항목이_없으면_400(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, authReq(t, "cust-1", http.MethodPost, "/orders", `{"items":[]}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("빈 주문은 400 여야 하는데 %d", rec.Code)
	}
}

func TestGetOrder_없는_주문이면_404(t *testing.T) {
	srv := newTestServer()
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, authReq(t, "cust-1", http.MethodGet, "/orders/order_none", ""))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("없는 주문은 404 여야 하는데 %d", rec.Code)
	}
}
