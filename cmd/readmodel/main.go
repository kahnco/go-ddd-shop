package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
	"github.com/kahnco/go-ddd-shop/internal/readmodel"
)

// readmodel 서비스: CQRS 의 읽기 쪽.
// 주문 이벤트(ordering.order.>)를 구독해 조회용 뷰를 만들고, "내 주문 목록"을 싸게 답한다.
// JetStream 내구 소비자라, 처음 붙으면 스트림의 과거 이벤트까지 재생해 뷰를 재구축한다.
func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	shutdown, _ := telemetry.InitTracer(context.Background(), "readmodel")
	defer func() { _ = shutdown(context.Background()) }()

	url := envOr("NATS_URL", "nats://localhost:4222")
	bus, err := eventbus.Connect(url, eventbus.OptionsFromEnv()...)
	if err != nil {
		logger.Error("nats 연결 실패", "url", url, "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	store := readmodel.NewMemoryStore()
	projector := readmodel.NewProjector(store, logger)

	// 주문의 모든 생명주기 이벤트를 하나의 구독으로 받는다.
	if err := bus.Subscribe("ordering.order.>", "readmodel", projector.Handle); err != nil {
		logger.Error("구독 실패", "err", err)
		os.Exit(1)
	}
	logger.Info("readmodel 서비스 시작 — ordering.order.> 구독 중", "nats", url)

	// 죽은 편지함 모니터: 재시도를 다 쓰고도 실패한 독성 메시지를 로그로 드러낸다.
	// (운영에선 여기에 알림·대시보드를 붙이고, 원인을 고친 뒤 bus.Redeliver 로 재투입한다.)
	if os.Getenv("NATS_JETSTREAM") != "" {
		if err := bus.SubscribeDLQ("readmodeldlq", func(dl eventbus.DeadLetter) error {
			logger.Warn("죽은 편지(DLQ) — 처리 포기된 이벤트",
				"subject", dl.Subject, "group", dl.Group,
				"attempts", dl.Attempts, "error", dl.Error, "event_id", dl.Event.ID)
			return nil
		}); err != nil {
			logger.Error("DLQ 모니터 구독 실패", "err", err)
		} else {
			logger.Info("DLQ 모니터 시작 — dlq.> 구독 중")
		}
	}

	mux := http.NewServeMux()
	readmodel.NewQueryHandler(store).Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("GET /metrics", telemetry.MetricsHandler())

	go func() {
		addr := envOr("HTTP_ADDR", ":8080")
		logger.Info("readmodel 조회 API", "addr", addr)
		if err := http.ListenAndServe(addr, telemetry.WrapHTTP(telemetry.Middleware(logger, mux), "readmodel")); err != nil {
			logger.Error("HTTP 서버 종료", "err", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("readmodel 서비스 종료")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
