// Package eventbus 는 bounded context 사이를 잇는 이벤트 버스다.
// 특정 도메인을 모르는 순수 전송 계층 — 이벤트 이름과 JSON 페이로드만 다룬다.
package eventbus

import "encoding/json"

// Envelope 는 브로커로 오가는 이벤트의 겉포장.
// Name 으로 "무슨 이벤트인지" 구분하고, Data 에 도메인 이벤트의 JSON 을 담는다.
// Meta 에는 상관 ID 처럼 도메인과 무관한 횡단 관심사(cross-cutting)를 싣는다 —
// 이 덕에 하나의 주문 흐름을 여러 서비스에 걸쳐 같은 ID 로 추적할 수 있다.
// 이 봉투 구조가 컨텍스트 간의 공통 계약이다(도메인 타입은 공유하지 않는다).
type Envelope struct {
	// ID 는 이 이벤트의 고유 식별자. 아웃박스가 재전송해도 같은 ID 를 유지하므로,
	// 소비자가 이 ID 로 중복을 걸러낼 수 있다(멱등성의 열쇠). 직접 발행 시엔 비어 있을 수 있다.
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	// SchemaVersion 은 Data 페이로드의 스키마 버전이다. 0/누락은 v1 로 본다.
	// 발행 시 최신 버전으로 찍히고, 소비 시 등록된 업캐스터로 최신까지 끌어올려진다.
	// 덕분에 JetStream 에 남아 있는 옛 버전 이벤트도 최신 소비자가 그대로 읽는다.
	SchemaVersion int               `json:"schema_version,omitempty"`
	Data          json.RawMessage   `json:"data"`
	Meta          map[string]string `json:"meta,omitempty"`
}

// NewEnvelope 는 payload 를 JSON 으로 직렬화해 봉투에 담는다.
func NewEnvelope(name string, payload any) (Envelope, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, err
	}
	return Envelope{Name: name, Data: data}, nil
}

// Into 는 봉투 안의 payload 를 target 으로 디코딩한다.
// 받는 컨텍스트는 자신만의 타입으로 풀어, 보내는 쪽 도메인에 의존하지 않는다.
func (e Envelope) Into(target any) error {
	return json.Unmarshal(e.Data, target)
}
