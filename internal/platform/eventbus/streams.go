package eventbus

import (
	"os"
	"strings"
	"time"

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
	{"CUSTOMER", []string{"customer.>"}},
	// 죽은 편지함(dead-letter). 재시도를 다 쓰고도 처리 못 한 독성 메시지가 여기 모인다.
	// 소비자를 막지 않으면서, 잃지도 않고, 나중에 사람이 보거나 재투입할 수 있게 보관한다.
	{"DLQ", []string{"dlq.>"}},
}

// ensureStreams 는 컨텍스트별 스트림을 만든다(이미 있으면 그대로 둔다).
// 모든 서비스가 시작 시 호출하므로, 기동 순서와 무관하게 스트림이 존재하게 된다.
func (b *Bus) ensureStreams() error {
	for _, s := range shopStreams {
		_, err := b.js.AddStream(&nats.StreamConfig{
			Name:       s.Name,
			Subjects:   s.Subjects,
			Storage:    nats.FileStorage,
			Duplicates: 2 * time.Minute, // Nats-Msg-Id 중복 제거 윈도우
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

type config struct {
	jetstream bool
	retry     *retryPolicy
}

// retryPolicy 는 처리 실패 시 몇 번, 어떤 간격으로 재전송할지 정한다.
// backoff 는 재전송 사이의 지연(지수적으로 늘림). maxDeliver 를 다 쓰면 DLQ 로 보낸다.
type retryPolicy struct {
	maxDeliver int
	backoff    []time.Duration
}

// defaultRetry: 최대 5회, 1s→2s→5s→10s 지연. 일시적 장애는 넘기고, 진짜 독성만 DLQ 로.
var defaultRetry = retryPolicy{
	maxDeliver: 5,
	backoff:    []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second},
}

// WithJetStream 은 JetStream(영속 스트림 + 내구 소비자)을 켠다.
func WithJetStream() Option { return func(c *config) { c.jetstream = true } }

// WithRetry 는 재시도 정책(최대 시도 횟수·백오프 지연)을 바꾼다.
// 테스트에서 짧은 백오프로 돌리거나, 서비스별로 인내심을 조절할 때 쓴다.
func WithRetry(maxDeliver int, backoff ...time.Duration) Option {
	return func(c *config) { c.retry = &retryPolicy{maxDeliver: maxDeliver, backoff: backoff} }
}

// OptionsFromEnv 는 NATS_JETSTREAM 이 설정돼 있으면 JetStream 을 켜는 옵션을 준다.
func OptionsFromEnv() []Option {
	if os.Getenv("NATS_JETSTREAM") != "" {
		return []Option{WithJetStream()}
	}
	return nil
}
