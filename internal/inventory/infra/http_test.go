package infra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStockHandler_재고를_돌려준다(t *testing.T) {
	repo := NewMemoryStockRepository()
	repo.Seed("prod-A", 7)
	mux := http.NewServeMux()
	NewStockHandler(repo).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stock/prod-A", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("상태 = 200 여야 하는데 %d", rec.Code)
	}
	var body struct {
		ProductID string `json:"product_id"`
		Available int    `json:"available"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.ProductID != "prod-A" || body.Available != 7 {
		t.Fatalf("응답 불일치: %+v", body)
	}
}

func TestStockHandler_없는_상품은_available_0(t *testing.T) {
	mux := http.NewServeMux()
	NewStockHandler(NewMemoryStockRepository()).Register(mux)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stock/prod-없음", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("상태 = 200 여야 하는데 %d", rec.Code)
	}
	var body struct {
		Available int `json:"available"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Available != 0 {
		t.Fatalf("없는 상품은 available=0 여야 하는데 %d", body.Available)
	}
}
