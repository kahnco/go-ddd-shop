package api

import (
	"context"
	"net/http"
)

// RegisterHealth 는 쿠버네티스 probe 용 엔드포인트 두 개를 건다.
//
//   - /healthz (liveness): 프로세스가 살아 있는가? 살아만 있으면 200.
//     실패하면 쿠버네티스가 파드를 재시작한다.
//   - /readyz (readiness): 트래픽을 받을 준비가 됐는가? 의존성(DB)까지 확인한다.
//     실패하면 Service 가 이 파드를 로드밸런싱에서 잠시 빼낸다(재시작은 안 함).
//
// 둘을 구분하는 게 핵심이다. DB 가 잠깐 끊겼을 때 재시작(liveness 실패)은 오히려
// 문제를 키우고, 트래픽만 잠시 안 주는 것(readiness 실패)이 옳다.
func RegisterHealth(mux *http.ServeMux, ready func(ctx context.Context) error) {
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := ready(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not ready", "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
}
