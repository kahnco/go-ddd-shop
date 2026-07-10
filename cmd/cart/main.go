package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kahnco/go-ddd-shop/internal/cart/app"
	"github.com/kahnco/go-ddd-shop/internal/cart/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// cart 서비스: 장바구니 담기·조회·결제.
// 결제(checkout)는 회원 서비스로 회원을 확인하고, 주문 서비스로 주문을 만든다(동기 호출).
// 그렇게 만들어진 주문이 기존 이벤트 사가(재고→결제→배송)를 그대로 탄다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	carts := infra.NewMemoryCartRepository()
	customers := infra.NewHTTPCustomerLookup(envOr("CUSTOMER_URL", "http://localhost:8085"))
	orders := infra.NewHTTPOrderPlacer(envOr("ORDERING_URL", "http://localhost:8080"))

	svc := app.NewCartService(carts, customers, orders)

	mux := http.NewServeMux()
	infra.NewCartHandler(svc).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	addr := envOr("HTTP_ADDR", ":8080")
	logger.Info("cart 서비스 시작", "addr", addr)
	if err := http.ListenAndServe(addr, telemetry.Middleware(logger, mux)); err != nil {
		logger.Error("서버 종료", "err", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
