package eventbus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Bus 는 NATS 로 뒷받침되는 이벤트 버스 어댑터.
// 발행/구독의 세부(직렬화·연결)를 숨기고, 위 계층에는 Envelope 만 노출한다.
type Bus struct {
	nc *nats.Conn
}

// Connect 는 NATS 서버에 연결한다. url 예: nats://localhost:4222
//
// RetryOnFailedConnect 로 서버가 아직 안 떠 있어도 즉시 실패하지 않고 배경에서 재접속한다.
// 컨테이너/쿠버네티스에서는 기동 순서를 보장하기 어려우니(브로커가 늦게 뜰 수 있다),
// 앱이 크래시 루프에 빠지지 않고 브로커를 기다리게 하는 편이 견고하다.
func Connect(url string) (*Bus, error) {
	nc, err := nats.Connect(url,
		nats.Name("go-ddd-shop"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats 연결: %w", err)
	}
	return &Bus{nc: nc}, nil
}

// Close 는 남은 메시지를 흘려보내고(drain) 연결을 닫는다.
func (b *Bus) Close() { _ = b.nc.Drain() }

// Publish 는 봉투를 subject 로 발행한다. Flush 로 실제 전송을 보장한다.
func (b *Bus) Publish(subject string, env Envelope) error {
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := b.nc.Publish(subject, raw); err != nil {
		return err
	}
	return b.nc.Flush()
}

// Handler 는 봉투 하나를 처리하는 콜백.
type Handler func(Envelope) error

// Subscribe 는 subject(와일드카드 가능)를 큐 그룹으로 구독한다.
// 같은 group 을 쓰는 인스턴스가 여럿이면 각 메시지는 그중 하나에만 전달된다
// (경쟁 소비 — 7편의 수평 확장에서 쓰인다).
func (b *Bus) Subscribe(subject, group string, handler Handler) error {
	_, err := b.nc.QueueSubscribe(subject, group, func(m *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(m.Data, &env); err != nil {
			return
		}
		_ = handler(env) // 핸들러가 자체적으로 에러를 로깅/처리한다
	})
	if err != nil {
		return err
	}
	return b.nc.Flush()
}
