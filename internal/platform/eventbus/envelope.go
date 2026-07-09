// Package eventbus 는 bounded context 사이를 잇는 이벤트 버스다.
// 특정 도메인을 모르는 순수 전송 계층 — 이벤트 이름과 JSON 페이로드만 다룬다.
package eventbus

import "encoding/json"

// Envelope 는 브로커로 오가는 이벤트의 겉포장.
// Name 으로 "무슨 이벤트인지" 구분하고, Data 에 도메인 이벤트의 JSON 을 담는다.
// 이 봉투 구조가 컨텍스트 간의 공통 계약이다(도메인 타입은 공유하지 않는다).
type Envelope struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
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
