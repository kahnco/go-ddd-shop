package eventbus

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// DeadLetter 는 재시도를 다 쓰고도 처리 못 한 이벤트의 기록이다.
// 원본 봉투에, 왜·몇 번 만에 죽었는지(실패 원인·시도 횟수·원래 subject)를 덧붙인다.
// 이 정보가 있어야 나중에 사람이 원인을 보거나, 고친 뒤 원래 자리로 재투입할 수 있다.
type DeadLetter struct {
	Subject  string   `json:"subject"`  // 원래 발행됐던 subject
	Group    string   `json:"group"`    // 어느 소비자 그룹이 처리에 실패했나
	Attempts int      `json:"attempts"` // 몇 번 시도했나
	Error    string   `json:"error"`    // 마지막 실패 원인
	Event    Envelope `json:"event"`    // 원본 봉투(그대로 재투입 가능)
}

// deliveryCount 는 이 메시지가 지금 몇 번째 전달인지(1-based) 돌려준다.
func deliveryCount(m *nats.Msg) int {
	if meta, err := m.Metadata(); err == nil {
		return int(meta.NumDelivered)
	}
	return 1
}

// publishDeadLetter 는 독성 메시지를 dlq.<subject> 로 보내 DLQ 스트림에 보관한다.
func (b *Bus) publishDeadLetter(subject, group string, attempts int, handlerErr error, env Envelope) error {
	dl := DeadLetter{
		Subject:  subject,
		Group:    group,
		Attempts: attempts,
		Error:    handlerErr.Error(),
		Event:    env,
	}
	raw, err := json.Marshal(dl)
	if err != nil {
		return err
	}
	var opts []nats.PubOpt
	if env.ID != "" {
		opts = append(opts, nats.MsgId("dlq-"+env.ID)) // 같은 이벤트가 두 번 죽어도 DLQ 에선 한 번만
	}
	_, err = b.js.Publish("dlq."+subject, raw, opts...)
	return err
}

// SubscribeDLQ 는 죽은 편지함을 구독한다. 대시보드·알림·재처리 도구가 여기에 붙어
// "무엇이, 왜 죽었는지"를 보고, 필요하면 Redeliver 로 되돌린다.
func (b *Bus) SubscribeDLQ(group string, handler func(DeadLetter) error) error {
	if b.js == nil {
		return fmt.Errorf("DLQ 는 JetStream 모드에서만 동작합니다")
	}
	_, err := b.js.QueueSubscribe("dlq.>", group, func(m *nats.Msg) {
		var dl DeadLetter
		if json.Unmarshal(m.Data, &dl) != nil {
			_ = m.Term()
			return
		}
		_ = handler(dl) // DLQ 는 종착지 — 모니터가 실패해도 메시지는 스트림에 남는다
		_ = m.Ack()
	},
		nats.Durable(durableName(group, "dlq.>")),
		nats.ManualAck(),
		nats.AckExplicit(),
	)
	return err
}

// Redeliver 는 죽은 편지를 원래 subject 로 다시 발행한다(원인을 고친 뒤 재투입).
func (b *Bus) Redeliver(dl DeadLetter) error {
	return b.Publish(dl.Subject, dl.Event)
}
