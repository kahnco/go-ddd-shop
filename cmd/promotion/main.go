package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// promotion 서비스: "정확히 N번째로 응모한 사람이 당첨"인 이벤트를 처리한다.
// 응모 입구는 HTTP(POST /events/{eventId}/entries)이고, 당첨이 확정되면
// promotion.winner_determined 를 발행한다(다운스트림이 상금 지급·알림).
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shutdown, _ := telemetry.InitTracer(context.Background(), "promotion")
	defer func() { _ = shutdown(context.Background()) }()

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	// 저장소 선택. DATABASE_URL 이 있으면 PostgreSQL(행 잠금으로 여러 replica 가 순번을
	// 빈틈없이 공유), 없으면 인메모리(뮤텍스로 단일 인스턴스 동시성은 안전).
	var repo app.Repository
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pg, err := infra.NewPostgresRepo(context.Background(), dsn)
		if err != nil {
			logger.Error("postgres 연결 실패", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		repo = pg
		logger.Info("응모 저장소 = PostgreSQL(다중 replica 안전, gapless)")
	} else {
		repo = infra.NewMemoryRepo()
		logger.Info("응모 저장소 = 인메모리(단일 인스턴스)")
	}

	publisher := infra.NewNatsEventPublisher(bus, "promotion")
	svc := app.NewService(repo, publisher)

	// 데모 이벤트: 시작 즉시, 정확히 TARGET 번째 당첨.
	target := atoiOr(os.Getenv("EVENT_TARGET"), 1000)
	eventID := envOr("EVENT_ID", "launch-2026")
	if err := svc.SeedEvent(context.Background(), domain.Event{
		ID: eventID, TargetSeq: target, StartsAt: time.Now(),
	}); err != nil {
		logger.Error("이벤트 시드 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("프로모션 이벤트 준비", "event", eventID, "target", target)

	mux := http.NewServeMux()
	infra.NewEntryHandler(svc).Register(mux) // POST /events/{eventId}/entries
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	httpAddr := envOr("HTTP_ADDR", ":8080")
	go func() {
		logger.Info("promotion 서비스 시작 — POST /events/{eventId}/entries", "addr", httpAddr)
		if err := http.ListenAndServe(httpAddr, mux); err != nil {
			logger.Error("HTTP 서버 종료", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("promotion 서비스 종료")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
