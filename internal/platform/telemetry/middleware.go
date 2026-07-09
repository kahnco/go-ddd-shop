package telemetry

import (
	"log/slog"
	"net/http"
	"time"
)

// statusRecorder 는 응답 상태코드를 가로채 기록한다(메트릭·로그용).
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware 는 모든 HTTP 요청에 대해 세 가지를 한다.
//   - 상관 ID 를 확보(헤더에 있으면 잇고, 없으면 생성)해 ctx·응답 헤더에 넣는다
//   - 처리 시간과 상태코드로 접근 로그를 남긴다
//   - http_requests_total 메트릭을 올린다
//
// 상관 ID 를 ctx 에 넣어 두면, 이 요청에서 비롯된 도메인 이벤트에도 그대로 실려
// 여러 서비스에 걸친 하나의 주문 흐름을 같은 ID 로 추적할 수 있다(분산 추적의 축소판).
func Middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cid := r.Header.Get(HeaderCorrelationID)
		if cid == "" {
			cid = NewID()
		}
		w.Header().Set(HeaderCorrelationID, cid)
		ctx := WithCorrelationID(r.Context(), cid)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		req := r.WithContext(ctx)
		start := time.Now()
		next.ServeHTTP(rec, req)
		elapsed := time.Since(start)

		// req.Pattern 은 ServeMux 가 매칭한 라우트 패턴(예: "POST /orders/{id}").
		// URL 경로 대신 이걸 라벨로 써야 주문 ID 마다 라벨이 폭증하는 걸 막는다(카디널리티).
		route := req.Pattern
		if route == "" {
			route = "unmatched"
		}
		httpRequests.WithLabelValues(r.Method, route, http.StatusText(rec.status)).Inc()
		logger.Info("http request",
			"method", r.Method, "path", r.URL.Path, "status", rec.status,
			"dur_ms", elapsed.Milliseconds(), "correlation_id", cid)
	})
}
