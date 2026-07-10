package eventbus

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// Bus 는 NATS 로 뒷받침되는 이벤트 버스 어댑터.
// core 모드(무영속)와 JetStream 모드(영속 스트림 + 내구 소비자)를 모두 지원한다.
// 위 계층에는 Envelope 만 노출하고, 어느 모드인지는 숨긴다.
type Bus struct {
	nc *nats.Conn
	js nats.JetStreamContext // nil 이면 core 모드
}

// Connect 는 NATS 서버에 연결한다. url 예: nats://localhost:4222
// WithJetStream() 을 주면 JetStream 을 켜고 컨텍스트별 스트림을 보장한다.
//
// RetryOnFailedConnect 로 서버가 아직 안 떠 있어도 즉시 실패하지 않고 배경에서 재접속한다.
func Connect(url string, opts ...Option) (*Bus, error) {
	var cfg config
	for _, o := range opts {
		o(&cfg)
	}

	nc, err := nats.Connect(url,
		nats.Name("go-ddd-shop"),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("nats 연결: %w", err)
	}

	b := &Bus{nc: nc}
	if cfg.jetstream {
		js, err := nc.JetStream()
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("jetstream 컨텍스트: %w", err)
		}
		b.js = js
		if err := b.ensureStreams(); err != nil {
			nc.Close()
			return nil, fmt.Errorf("스트림 보장: %w", err)
		}
	}
	return b, nil
}

// Close 는 남은 메시지를 흘려보내고(drain) 연결을 닫는다.
func (b *Bus) Close() { _ = b.nc.Drain() }

// Publish 는 봉투를 subject 로 발행한다.
// JetStream 모드에선 스트림에 영속 저장되고, core 모드에선 그냥 흘려보낸다.
func (b *Bus) Publish(subject string, env Envelope) error {
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if b.js != nil {
		_, err := b.js.Publish(subject, raw) // 발행 = 스트림에 저장(ack 까지 대기)
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
//
// JetStream 모드: 내구 소비자(durable)를 만든다. 소비자가 자기 위치를 기억하므로
// 재시작해도 놓치지 않고, 처리 성공 시 ack / 실패 시 nak(재전송)한다. 재전송은 중복을
// 부르니 소비자는 멱등해야 한다(봉투 ID 로 중복 제거 — 없으면 스트림 시퀀스를 ID 로 채운다).
//
// core 모드: QueueSubscribe(무영속, at-most-once).
func (b *Bus) Subscribe(subject, group string, handler Handler) error {
	if b.js != nil {
		_, err := b.js.QueueSubscribe(subject, group, func(m *nats.Msg) {
			var env Envelope
			if json.Unmarshal(m.Data, &env) != nil {
				_ = m.Term() // 파싱 불가 메시지는 버린다(재전송해도 소용없음)
				return
			}
			if env.ID == "" {
				if meta, err := m.Metadata(); err == nil {
					env.ID = fmt.Sprintf("js-%d", meta.Sequence.Stream) // 재전송돼도 같은 값 → 멱등 키
				}
			}
			if err := handler(env); err != nil {
				_ = m.Nak() // 처리 실패 → 재전송 요청
				return
			}
			_ = m.Ack()
		},
			nats.Durable(durableName(group, subject)),
			nats.ManualAck(),
			nats.AckExplicit(),
		)
		return err
	}

	_, err := b.nc.QueueSubscribe(subject, group, func(m *nats.Msg) {
		var env Envelope
		if json.Unmarshal(m.Data, &env) == nil {
			_ = handler(env)
		}
	})
	if err != nil {
		return err
	}
	return b.nc.Flush()
}
