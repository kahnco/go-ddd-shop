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
)

// RecordEventPublished/Consumed 는 이벤트 계측용. 발행·소비 어댑터가 호출한다.
func RecordEventPublished(event string)        { eventsPublished.WithLabelValues(event).Inc() }
func RecordEventConsumed(event, result string) { eventsConsumed.WithLabelValues(event, result).Inc() }

// MetricsHandler 는 /metrics 엔드포인트 핸들러(프로메테우스가 스크레이프한다).
func MetricsHandler() http.Handler { return promhttp.Handler() }
