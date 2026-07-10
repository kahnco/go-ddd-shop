package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/kahnco/go-ddd-shop/internal/catalog/app"
	"github.com/kahnco/go-ddd-shop/internal/catalog/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// catalog 서비스: 상품과 가격의 소유자.
// HTTP 로 상품을 조회·등록·가격변경하고, 변경 사실을 이벤트로 발행한다.
// 주문 서비스는 부트스트랩 때 GET /products 로 현재 카탈로그를 읽고,
// 이후 catalog.* 이벤트로 가격 프로젝션을 최신으로 유지한다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := context.Background()

	repo := infra.NewMemoryProductRepository()

	// 발행 어댑터: NATS 있으면 브로커로, 없으면 로그(단독 실행).
	var publisher app.EventPublisher = infra.NewLogPublisher(logger)
	if url := os.Getenv("NATS_URL"); url != "" {
		bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
		if err != nil {
			logger.Error("nats 연결 실패", "url", url, "err", err)
			os.Exit(1)
		}
		defer bus.Close()
		publisher = infra.NewNatsEventPublisher(bus, "catalog")
		logger.Info("이벤트 발행 = NATS", "url", url)
	}

	svc := app.NewCatalogService(repo, publisher)

	// 데모용 초기 상품. 등록과 동시에 ProductAdded 가 발행돼 주문 프로젝션이 채워진다.
	_ = svc.AddProduct(ctx, "prod-A", "사과", 1000)
	_ = svc.AddProduct(ctx, "prod-B", "바나나", 3000)

	mux := http.NewServeMux()
	infra.NewCatalogHandler(svc).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	addr := envOr("HTTP_ADDR", ":8080")
	logger.Info("catalog 서비스 시작", "addr", addr)
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
