package telemetry

import (
	"context"
	"net/http"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracer 는 OpenTelemetry 추적을 켠다.
//   - W3C traceparent 전파기를 항상 설정한다(HTTP·이벤트 경계로 trace 를 잇기 위해).
//   - OTEL_EXPORTER_OTLP_ENDPOINT 가 있으면 OTLP(gRPC)로 span 을 내보낸다(예: Jaeger).
//     없으면 내보내진 않되, trace/span ID 는 여전히 생성·전파된다.
//   - 아웃바운드 HTTP(DefaultTransport)를 계측해, 서비스 간 HTTP 호출에 trace 를 전파한다.
func InitTracer(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	http.DefaultTransport = otelhttp.NewTransport(http.DefaultTransport)

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil // 내보내기 없이도 ID 는 흐른다
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}
	res, _ := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(serviceName)))
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}

// WrapHTTP 는 HTTP 핸들러를 계측한다 — 들어온 traceparent 를 추출하고 서버 span 을 연다.
func WrapHTTP(handler http.Handler, name string) http.Handler {
	return otelhttp.NewHandler(handler, name)
}

// Tracer 는 이 앱의 이름 있는 트레이서.
func Tracer() trace.Tracer { return otel.Tracer("go-ddd-shop") }

// StartSpan 은 새 span 을 연다. 반드시 span.End() 로 닫아야 한다.
func StartSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name)
}

// MetaFromContext 는 이벤트 봉투에 실을 메타(상관 ID + W3C traceparent)를 만든다.
// 발행 어댑터가 호출한다 — 이 메타가 소비 서비스의 span 을 producer 의 trace 에 잇는다.
func MetaFromContext(ctx context.Context) map[string]string {
	m := map[string]string{}
	if cid := CorrelationID(ctx); cid != "" {
		m[MetaCorrelationID] = cid
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(m))
	if len(m) == 0 {
		return nil
	}
	return m
}

// ContextFromMeta 는 봉투 메타에서 상관 ID·trace 컨텍스트를 꺼내 ctx 에 복원한다.
// 소비 어댑터가 호출한다.
func ContextFromMeta(ctx context.Context, m map[string]string) context.Context {
	if cid := m[MetaCorrelationID]; cid != "" {
		ctx = WithCorrelationID(ctx, cid)
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(m))
}

// TraceparentFromContext 는 현재 trace 컨텍스트를 문자열로 뽑는다(아웃박스에 저장용).
func TraceparentFromContext(ctx context.Context) string {
	m := map[string]string{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(m))
	return m["traceparent"]
}

// ContextWithTraceparent 는 저장해 둔 traceparent 로부터 trace 컨텍스트를 복원한다(아웃박스 릴레이용).
func ContextWithTraceparent(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(map[string]string{"traceparent": traceparent}))
}
