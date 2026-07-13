package telemetry

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus 메트릭. 기본 레지스트리에 등록되고 /metrics 로 노출된다.
var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "처리한 HTTP 요청 수",
	}, []string{"method", "path", "status"})

	eventsPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_published_total",
		Help: "브로커로 발행한 도메인 이벤트 수",
	}, []string{"event"})

	eventsConsumed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_consumed_total",
		Help: "브로커에서 소비한 이벤트 수",
	}, []string{"event", "result"})

	// RED 의 Duration. 라벨은 method·route 만(status 는 빼 카디널리티를 낮춘다).
	// 버킷은 웹 API 지연에 맞춰 5ms~5s. 히스토그램이라 p50/p95/p99 를 쿼리로 뽑을 수 있다.
	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP 요청 처리 시간(초)",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"method", "route"})

	// 비즈니스 메트릭 — 기술 지표만으론 "장사가 되는가"를 못 본다.
	ordersPlaced = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "orders_placed_total",
		Help: "생성된 주문 수(유입 경로별)",
	}, []string{"channel"})

	orderRevenue = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "order_revenue_won_total",
		Help: "생성된 주문 총액(원, 유입 경로별)",
	}, []string{"channel"})

	// DLQ 유입 — 14편의 죽은 편지함이 얼마나 쌓이는지. 시스템 건강의 핵심 지표.
	dlqMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dlq_messages_total",
		Help: "죽은 편지함(DLQ)으로 보낸 이벤트 수",
	}, []string{"subject"})
)

// RecordEventPublished/Consumed 는 이벤트 계측용. 발행·소비 어댑터가 호출한다.
func RecordEventPublished(event string)        { eventsPublished.WithLabelValues(event).Inc() }
func RecordEventConsumed(event, result string) { eventsConsumed.WithLabelValues(event, result).Inc() }

// RecordHTTPDuration 은 요청 처리 시간을 기록한다(미들웨어가 호출).
func RecordHTTPDuration(method, route string, seconds float64) {
	httpDuration.WithLabelValues(method, route).Observe(seconds)
}

// RecordOrderPlaced 는 주문 생성을 비즈니스 지표로 기록한다(주문 유스케이스가 호출).
func RecordOrderPlaced(channel string, amountWon int64) {
	if channel == "" {
		channel = "web"
	}
	ordersPlaced.WithLabelValues(channel).Inc()
	orderRevenue.WithLabelValues(channel).Add(float64(amountWon))
}

// RecordDeadLettered 는 DLQ 유입을 기록한다(이벤트 버스가 호출).
func RecordDeadLettered(subject string) { dlqMessages.WithLabelValues(subject).Inc() }

// MetricsHandler 는 /metrics 엔드포인트 핸들러(프로메테우스가 스크레이프한다).
func MetricsHandler() http.Handler { return promhttp.Handler() }
