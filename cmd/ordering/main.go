package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/kahnco/go-ddd-shop/internal/ordering/api"
	"github.com/kahnco/go-ddd-shop/internal/ordering/app"
	"github.com/kahnco/go-ddd-shop/internal/ordering/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

// main 은 "조립 루트(composition root)".
// 여기서만 구체 어댑터를 골라 포트에 끼운다. 도메인·애플리케이션 코드는
// 이 파일이 무엇을 고르는지 전혀 모른다 — 그래서 나중에 저장소를 PostgreSQL 로,
// 발행을 NATS 로 바꿔도 이 파일 몇 줄만 바뀐다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	repo := infra.NewMemoryOrderRepository()
	ids := infra.RandomIDGenerator{}

	// 발행 어댑터 선택. NATS_URL 이 있으면 브로커로 발행하고, 없으면 로그로 찍는다.
	// 유스케이스(OrderService)는 EventPublisher 포트만 보므로, 여기서 무엇을 끼우든 모른다.
	var publisher app.EventPublisher = infra.NewLogPublisher(logger)
	if url := os.Getenv("NATS_URL"); url != "" {
		bus, err := eventbus.Connect(url)
		if err != nil {
			logger.Error("nats 연결 실패", "url", url, "err", err)
			os.Exit(1)
		}
		defer bus.Close()
		publisher = infra.NewNatsEventPublisher(bus, "ordering")
		logger.Info("이벤트 발행 = NATS", "url", url)
	}

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
