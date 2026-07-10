package idempotency

import "testing"

func TestGuard_같은ID는_한번만_실행된다(t *testing.T) {
	g := NewGuard()
	count := 0
	fn := func() error { count++; return nil }

	_ = g.Do("evt-1", fn)
	_ = g.Do("evt-1", fn) // 중복
	_ = g.Do("evt-1", fn) // 중복

	if count != 1 {
		t.Fatalf("같은 ID 는 한 번만 실행돼야 하는데 %d회", count)
	}
}

func TestGuard_다른ID는_각각_실행된다(t *testing.T) {
	g := NewGuard()
	count := 0
	fn := func() error { count++; return nil }

	_ = g.Do("evt-1", fn)
	_ = g.Do("evt-2", fn)

	if count != 2 {
		t.Fatalf("다른 ID 는 각각 실행돼야 하는데 %d회", count)
	}
}

func TestGuard_빈ID는_매번_실행된다(t *testing.T) {
	g := NewGuard()
	count := 0
	_ = g.Do("", func() error { count++; return nil })
	_ = g.Do("", func() error { count++; return nil })
	if count != 2 {
		t.Fatalf("빈 ID 는 매번 실행돼야 하는데 %d회", count)
	}
}

func TestGuard_실패는_기록하지않아_재시도가능(t *testing.T) {
	g := NewGuard()
	attempts := 0
	failing := func() error { attempts++; return errBoom }

	_ = g.Do("evt-1", failing)                                 // 실패
	_ = g.Do("evt-1", func() error { attempts++; return nil }) // 재시도 → 실행돼야 함

	if attempts != 2 {
		t.Fatalf("실패 후 재시도가 실행돼야 하는데 attempts=%d", attempts)
	}
}

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }
