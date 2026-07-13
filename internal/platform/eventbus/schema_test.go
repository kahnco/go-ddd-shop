package eventbus

import (
	"encoding/json"
	"testing"
)

// 더하기 변경(필드 추가)은 관용적 리더가 그냥 소화한다 — 업캐스터 없이도.
func TestTolerantReader_모르는_필드는_무시_빠진_필드는_기본값(t *testing.T) {
	// v2 생산자가 보낸(새 필드 channel 포함) 이벤트를, 그 필드를 모르는 옛 소비자가 읽는다.
	rawV2 := `{"order_id":"o1","channel":"app","extra":"미래필드"}`
	var oldConsumer struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(rawV2), &oldConsumer); err != nil {
		t.Fatalf("옛 소비자가 새 이벤트를 못 읽으면 안 됨: %v", err)
	}
	if oldConsumer.OrderID != "o1" {
		t.Errorf("order_id = o1 여야: %s", oldConsumer.OrderID)
	}

	// 반대로, v1 이벤트(channel 없음)를 새 소비자가 읽으면 channel 은 zero-value("").
	rawV1 := `{"order_id":"o1"}`
	var newConsumer struct {
		OrderID string `json:"order_id"`
		Channel string `json:"channel"`
	}
	_ = json.Unmarshal([]byte(rawV1), &newConsumer)
	if newConsumer.Channel != "" {
		t.Errorf("빠진 필드는 기본값이어야: %q", newConsumer.Channel)
	}
}

// 깨는 변경은 업캐스터로. 여기선 "v1(채널 없음) → v2(channel=web 채움)" 를 검증한다.
func TestUpcast_옛_버전을_최신으로_끌어올린다(t *testing.T) {
	// 테스트 격리를 위해 레지스트리를 저장/복원.
	savedUp, savedLatest := upcasters, latestSchema
	upcasters, latestSchema = map[string]map[int]Upcaster{}, map[string]int{}
	defer func() { upcasters, latestSchema = savedUp, savedLatest }()

	RegisterUpcaster("order.placed", 1, func(e Envelope) Envelope {
		var m map[string]json.RawMessage
		_ = json.Unmarshal(e.Data, &m)
		if _, ok := m["channel"]; !ok {
			m["channel"], _ = json.Marshal("web")
			e.Data, _ = json.Marshal(m)
		}
		return e
	})

	if latestVersion("order.placed") != 2 {
		t.Fatalf("최신 버전 = 2 여야: %d", latestVersion("order.placed"))
	}

	// 버전 없는 옛 이벤트(v0 → v1 취급).
	old := Envelope{Name: "order.placed", Data: json.RawMessage(`{"order_id":"o1"}`)}
	got := upcast(old)

	if got.SchemaVersion != 2 {
		t.Errorf("업캐스트 후 버전 = 2 여야: %d", got.SchemaVersion)
	}
	var m map[string]string
	_ = json.Unmarshal(got.Data, &m)
	if m["channel"] != "web" {
		t.Errorf("옛 이벤트에 channel=web 이 채워져야: %q", m["channel"])
	}
}

// 이미 최신인(channel 있는) v2 이벤트는 업캐스터가 건드리지 않는다(멱등).
func TestUpcast_최신_이벤트는_그대로_둔다(t *testing.T) {
	savedUp, savedLatest := upcasters, latestSchema
	upcasters, latestSchema = map[string]map[int]Upcaster{}, map[string]int{}
	defer func() { upcasters, latestSchema = savedUp, savedLatest }()

	RegisterUpcaster("order.placed", 1, func(e Envelope) Envelope {
		var m map[string]json.RawMessage
		_ = json.Unmarshal(e.Data, &m)
		if _, ok := m["channel"]; !ok {
			m["channel"], _ = json.Marshal("web")
			e.Data, _ = json.Marshal(m)
		}
		return e
	})

	v2 := Envelope{Name: "order.placed", SchemaVersion: 2, Data: json.RawMessage(`{"order_id":"o1","channel":"app"}`)}
	got := upcast(v2)
	var m map[string]string
	_ = json.Unmarshal(got.Data, &m)
	if m["channel"] != "app" {
		t.Errorf("이미 최신인 이벤트의 channel 이 바뀌면 안 됨: %q", m["channel"])
	}
}
