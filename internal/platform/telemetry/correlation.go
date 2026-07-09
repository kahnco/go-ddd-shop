// Package telemetry 는 관찰성(observability)의 공통 조각을 모은다 —
// 상관 ID 전파, 메트릭, HTTP 미들웨어. 특정 도메인을 모르는 순수 인프라 계층이다.
package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

// HeaderCorrelationID 는 상관 ID 를 실어 나르는 HTTP 헤더 이름.
const HeaderCorrelationID = "X-Correlation-ID"

// MetaCorrelationID 는 이벤트 봉투 메타데이터에서 상관 ID 를 담는 키.
const MetaCorrelationID = "correlation_id"

type ctxKey int

const correlationKey ctxKey = 0

// NewID 는 새 상관 ID 를 만든다. 요청 하나(그리고 그로 인한 이벤트들)를 관통하는 꼬리표다.
func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithCorrelationID 는 ctx 에 상관 ID 를 심는다.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey, id)
}

// CorrelationID 는 ctx 에서 상관 ID 를 꺼낸다. 없으면 빈 문자열.
func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationKey).(string); ok {
		return id
	}
	return ""
}
