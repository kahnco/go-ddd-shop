package eventbus

import (
	"os"
	"strings"

	"github.com/nats-io/nats.go"
)

// shopStreams 는 "컨텍스트별 스트림" 정의다. 각 bounded context 가 자기 subject 대역을
// 하나의 스트림으로 소유한다. 스트림에 저장된 이벤트는 소비자가 늦게 붙거나 잠깐 죽어도
// 사라지지 않으니, core NATS 의 at-most-once(구독 전 발행 유실) 문제가 해결된다.
var shopStreams = []struct {
	Name     string
	Subjects []string
}{
	{"ORDERING", []string{"ordering.>"}},
	{"INVENTORY", []string{"inventory.>"}},
	{"PAYMENT", []string{"payment.>"}},
	{"SHIPPING", []string{"shipping.>"}},
	{"CATALOG", []string{"catalog.>"}},
}

// ensureStreams 는 컨텍스트별 스트림을 만든다(이미 있으면 그대로 둔다).
// 모든 서비스가 시작 시 호출하므로, 기동 순서와 무관하게 스트림이 존재하게 된다.
func (b *Bus) ensureStreams() error {
	for _, s := range shopStreams {
		_, err := b.js.AddStream(&nats.StreamConfig{
			Name:     s.Name,
			Subjects: s.Subjects,
			Storage:  nats.FileStorage,
		})
		if err != nil && !strings.Contains(err.Error(), "already in use") {
			return err
		}
	}
	return nil
}

// durableName 은 (그룹, subject)로부터 안정적인 내구 소비자 이름을 만든다.
// 내구 소비자 이름엔 . * > 공백을 쓸 수 없어 치환한다.
func durableName(group, subject string) string {
	repl := strings.NewReplacer(".", "_", "*", "star", ">", "all", " ", "_")
	return group + "_" + repl.Replace(subject)
}

// Option 은 Connect 의 동작을 바꾼다.
type Option func(*config)

type config struct{ jetstream bool }

// WithJetStream 은 JetStream(영속 스트림 + 내구 소비자)을 켠다.
func WithJetStream() Option { return func(c *config) { c.jetstream = true } }

// OptionsFromEnv 는 NATS_JETSTREAM 이 설정돼 있으면 JetStream 을 켜는 옵션을 준다.
func OptionsFromEnv() []Option {
	if os.Getenv("NATS_JETSTREAM") != "" {
		return []Option{WithJetStream()}
	}
	return nil
}
