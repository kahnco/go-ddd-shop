package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/ratelimit"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
	"github.com/kahnco/go-ddd-shop/internal/promotion/infra"
)

// promotion 서비스: "정확히 N번째로 응모한 사람이 당첨"인 이벤트를 처리한다.
//
//   - 순번 배정: 빈틈없는(gapless) 카운터(인메모리 뮤텍스 / Postgres 행 잠금)
//   - 당첨 통지: 아웃박스에 같은 트랜잭션으로 적재 → 릴레이가 발행(정확히-한번 지향)
//   - 접수: 동기(즉시 순번) 또는 큐 완충(202 접수 → 순차 소비자가 배정) — INGEST 로 선택
//   - 어뷰징 방어: 사용자/IP 별 토큰 버킷 레이트 리밋(429)
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown, _ := telemetry.InitTracer(context.Background(), "promotion")
	defer func() { _ = shutdown(context.Background()) }()

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	// 저장소 선택. DATABASE_URL 이 있으면 PostgreSQL(행 잠금·여러 replica·아웃박스),
	// 없으면 인메모리(단일 인스턴스).
	var repo app.Repository
	var store infra.OutboxStore
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		pg, err := infra.NewPostgresRepo(context.Background(), dsn)
		if err != nil {
			logger.Error("postgres 연결 실패", "err", err)
			os.Exit(1)
		}
		defer pg.Close()
		repo, store = pg, pg
		logger.Info("응모 저장소 = PostgreSQL(다중 replica·gapless·아웃박스)")
	} else {
		mem := infra.NewMemoryRepo()
		repo, store = mem, mem
		logger.Info("응모 저장소 = 인메모리(단일 인스턴스)")
	}

	svc := app.NewService(repo)

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

	// 아웃박스 릴레이 — 당첨 이벤트를 브로커로 흘린다(커밋됐으면 반드시 발행).
	relay := infra.NewOutboxRelay(store, bus, 500*time.Millisecond, logger)
	go relay.Run(ctx)

	// 레이트 리밋 — 사용자(X-User-Id) 우선, 없으면 IP. 버스트/속도는 환경변수로.
	// REDIS_URL 이 있으면 Redis 분산 리밋(여러 replica 공유), 없으면 인메모리(단일 인스턴스).
	burst := atoiOr(os.Getenv("RATE_BURST"), 5)
	perSec := float64(atoiOr(os.Getenv("RATE_PER_SEC"), 2))
	rateKey := func(r *http.Request) string {
		if u := r.Header.Get("X-User-Id"); u != "" {
			return u
		}
		return ratelimit.ClientIP(r)
	}
	var rateMiddleware func(http.Handler) http.Handler
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			logger.Error("REDIS_URL 파싱 실패", "err", err)
			os.Exit(1)
		}
		rateMiddleware = ratelimit.NewRedis(redis.NewClient(opt), burst, perSec).Middleware(rateKey)
		logger.Info("레이트 리밋 = Redis 분산", "url", redisURL)
	} else {
		rateMiddleware = ratelimit.New(burst, perSec).Middleware(rateKey)
		logger.Info("레이트 리밋 = 인메모리(단일 인스턴스)")
	}

	mux := http.NewServeMux()

	// 접수 모드 — 동기(기본) 또는 큐 완충.
	entryMux := http.NewServeMux()
	if envOr("INGEST", "sync") == "queue" {
		hub := infra.NewHub()
		infra.NewQueuedEntryHandler(infra.NewNatsEntryQueue(bus)).Register(entryMux)
		consumer := infra.NewEntryRequestedConsumer(svc, logger).WithHub(hub)
		if err := bus.Subscribe(infra.EntryRequestedSubject, "promotion", consumer.Handle); err != nil {
			logger.Error("entry_requested 구독 실패", "err", err)
			os.Exit(1)
		}
		infra.NewSSEHandler(svc, hub).Register(mux) // GET …/entries/me/stream — 결과 실시간 push
		logger.Info("접수 모드 = 큐 완충(202 접수 → 순차 배정 → SSE push)")
	} else {
		infra.NewEntryHandler(svc).Register(entryMux)
		logger.Info("접수 모드 = 동기(즉시 순번)")
	}
	// POST 응모에만 레이트 리밋을 씌운다.
	mux.Handle("POST /events/{eventId}/entries", rateMiddleware(entryMux))

	// 상태 조회(큐 모드에서 결과 확인) — 레이트 리밋 없이.
	infra.NewStatusHandler(svc).Register(mux)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	httpAddr := envOr("HTTP_ADDR", ":8080")
	go func() {
		logger.Info("promotion 서비스 시작", "addr", httpAddr)
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
