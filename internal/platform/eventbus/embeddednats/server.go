// Package embeddednats 는 인메모리 NATS 서버를 띄우는 헬퍼다.
// 테스트와 로컬 데모에서 외부 브로커 설치 없이 이벤트 흐름을 그대로 돌려볼 수 있게 한다.
// (임베디드 서버라도 TCP 위의 진짜 NATS 프로토콜을 쓰므로, 실제 통신 경로를 검증한다.)
// JetStream 을 켜 두어, core 모드와 JetStream 모드 테스트를 모두 지원한다.
package embeddednats

import (
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats-server/v2/server"
)

// Start 는 임의의 빈 포트로 NATS 서버(JetStream 포함)를 띄우고, 접속 URL 과 종료 함수를 돌려준다.
func Start() (url string, shutdown func(), err error) {
	dir, err := os.MkdirTemp("", "nats-js-")
	if err != nil {
		return "", nil, err
	}
	opts := &server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
		JetStream: true,
		StoreDir:  dir,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		os.RemoveAll(dir)
		return "", nil, err
	}
	go ns.Start()
	if !ns.ReadyForConnections(5 * time.Second) {
		ns.Shutdown()
		os.RemoveAll(dir)
		return "", nil, fmt.Errorf("nats 서버가 시간 안에 준비되지 않음")
	}
	return ns.ClientURL(), func() {
		ns.Shutdown()
		os.RemoveAll(dir)
	}, nil
}
