package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/customer/app"
	"github.com/kahnco/go-ddd-shop/internal/customer/infra"
	"github.com/kahnco/go-ddd-shop/internal/platform/auth"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// customer 서비스: 회원 등록·조회. 장바구니가 결제 시 회원 존재를 여기서 확인한다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shutdown, _ := telemetry.InitTracer(context.Background(), "customer")
	defer func() { _ = shutdown(context.Background()) }()

	repo := infra.NewMemoryCustomerRepository()

	var publisher app.EventPublisher = infra.NewLogPublisher(logger)
	if url := os.Getenv("NATS_URL"); url != "" {
		bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
		if err != nil {
			logger.Error("nats 연결 실패", "url", url, "err", err)
			os.Exit(1)
		}
		defer bus.Close()
		publisher = infra.NewNatsEventPublisher(bus)
		logger.Info("이벤트 발행 = NATS")
	}

	// 인증 조립: 비밀번호는 bcrypt 로 해시하고, 로그인 성공 시 24시간짜리 JWT 를 발급한다.
	secret := auth.SecretFromEnv(logger)
	svc := app.NewCustomerService(
		repo, publisher,
		infra.NewBcryptHasher(),
		infra.NewJWTIssuer(secret, 24*time.Hour),
		infra.RandomIDGenerator{},
	)

	mux := http.NewServeMux()
	infra.NewCustomerHandler(svc).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	addr := envOr("HTTP_ADDR", ":8080")
	logger.Info("customer 서비스 시작", "addr", addr)
	if err := http.ListenAndServe(addr, telemetry.WrapHTTP(telemetry.Middleware(logger, mux), "customer")); err != nil {
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
