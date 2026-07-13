// Package integration 은 컨텍스트를 가로지르는 배선(이벤트 스키마 업캐스터 등)을 담는다.
package integration

import (
	"encoding/json"

	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

// RegisterUpcasters 는 이벤트 스키마 진화 업캐스터를 등록한다.
// 모든 서비스가 기동 시(구독을 걸기 전에) 한 번 호출한다.
// 그래야 JetStream 스트림에 남아 있는 옛 버전 이벤트를 최신 소비자가 그대로 읽는다.
func RegisterUpcasters() {
	// order.placed v1 → v2: v1 이벤트엔 channel 필드가 없다.
	// 없으면 "web" 으로 정규화해, 모든 소비자가 늘 channel 을 볼 수 있게 한다.
	// (멱등하게 작성 — 이미 channel 이 있으면 손대지 않는다.)
	eventbus.RegisterUpcaster("order.placed", 1, func(e eventbus.Envelope) eventbus.Envelope {
		var m map[string]json.RawMessage
		if json.Unmarshal(e.Data, &m) != nil {
			return e
		}
		if _, ok := m["channel"]; !ok {
			m["channel"], _ = json.Marshal("web")
			if data, err := json.Marshal(m); err == nil {
				e.Data = data
			}
		}
		return e
	})
}
