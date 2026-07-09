package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kahnco/go-ddd-shop/internal/ordering/api"
	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
)

// main 은 "조립 루트(composition root)".
// 여기서만 구체 어댑터를 골라 포트에 끼운다. 도메인·애플리케이션 코드는
// 이 파일이 무엇을 고르는지 전혀 모른다 — 그래서 나중에 저장소를 PostgreSQL 로,
// 발행을 NATS 로 바꿔도 이 파일 몇 줄만 바뀐다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := infra.NewMemoryOrderRepository()
	publisher := infra.NewLogPublisher(logger)
	ids := infra.RandomIDGenerator{}

	svc := app.NewOrderService(repo, publisher, ids)

	mux := http.NewServeMux()
	api.NewOrderHandler(svc).Register(mux)

	addr := ":8080"
	logger.Info("ordering service 시작", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Error("서버 종료", "err", err)
		os.Exit(1)
	}
}
