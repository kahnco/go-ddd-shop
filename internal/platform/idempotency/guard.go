// Package idempotency 는 "같은 이벤트를 두 번 처리해도 한 번 처리한 것과 같게" 만든다.
// 아웃박스·브로커의 at-least-once 전달은 중복을 부르므로, 소비자는 멱등해야 한다.
package idempotency

import "sync"

// Guard 는 이미 처리한 이벤트 ID 를 기억해 중복 처리를 막는다.
// 데모용 인메모리 구현 — 실서비스에서는 이 기록을 DB(예: processed_events 테이블)에 두고,
// 이상적으로는 비즈니스 쓰기와 같은 트랜잭션에서 함께 커밋한다.
type Guard struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

func NewGuard() *Guard {
	return &Guard{seen: make(map[string]struct{})}
}

// Do 는 id 가 처음 보는 것일 때만 fn 을 실행한다.
// 이미 처리한 id 면 아무것도 하지 않고 성공으로 친다(중복 무시).
// id 가 비어 있으면(식별 불가) 매번 실행한다.
func (g *Guard) Do(id string, fn func() error) error {
	if id != "" {
		g.mu.Lock()
		_, done := g.seen[id]
		g.mu.Unlock()
		if done {
			return nil
		}
	}
	if err := fn(); err != nil {
		return err // 실패는 기록하지 않는다 — 재시도로 다시 처리될 수 있게
	}
	if id != "" {
		g.mu.Lock()
		g.seen[id] = struct{}{}
		g.mu.Unlock()
	}
	return nil
}
