package telemetry

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestRecordOrderPlaced_카운터와_매출을_올린다(t *testing.T) {
	before := testutil.ToFloat64(ordersPlaced.WithLabelValues("app"))
	beforeRev := testutil.ToFloat64(orderRevenue.WithLabelValues("app"))

	RecordOrderPlaced("app", 5000)

	if got := testutil.ToFloat64(ordersPlaced.WithLabelValues("app")); got != before+1 {
		t.Errorf("orders_placed_total(app) = %v, 원했던 값 %v", got, before+1)
	}
	if got := testutil.ToFloat64(orderRevenue.WithLabelValues("app")); got != beforeRev+5000 {
		t.Errorf("order_revenue_won_total(app) = %v, 원했던 값 %v", got, beforeRev+5000)
	}
}

func TestRecordOrderPlaced_빈_채널은_web으로(t *testing.T) {
	before := testutil.ToFloat64(ordersPlaced.WithLabelValues("web"))
	RecordOrderPlaced("", 1000) // 빈 채널
	if got := testutil.ToFloat64(ordersPlaced.WithLabelValues("web")); got != before+1 {
		t.Errorf("빈 채널은 web 으로 집계돼야: %v", got)
	}
}

func TestRecordDeadLettered_DLQ_카운터를_올린다(t *testing.T) {
	before := testutil.ToFloat64(dlqMessages.WithLabelValues("payment.completed"))
	RecordDeadLettered("payment.completed")
	if got := testutil.ToFloat64(dlqMessages.WithLabelValues("payment.completed")); got != before+1 {
		t.Errorf("dlq_messages_total = %v, 원했던 값 %v", got, before+1)
	}
}
