package auth

import (
	"testing"
)

func TestSecretFromEnv_주입된값을_쓴다(t *testing.T) {
	t.Setenv("JWT_SECRET", "a-real-strong-secret-32bytes-abcdefgh")
	if got := SecretFromEnv(nil); got != "a-real-strong-secret-32bytes-abcdefgh" {
		t.Fatalf("주입값을 써야: %q", got)
	}
}

func TestSecretFromEnv_비운영에서는_기본값으로_떨어진다(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	t.Setenv("APP_ENV", "")
	if got := SecretFromEnv(nil); got != devSecret {
		t.Fatalf("비운영에선 기본값이어야: %q", got)
	}
}

func TestSecretFromEnv_운영에서_기본값이면_부팅을_거부한다(t *testing.T) {
	// exit 훅을 가로채 os.Exit 대신 기록만 한다.
	var code int
	called := false
	orig := exit
	exit = func(c int) { code = c; called = true }
	defer func() { exit = orig }()

	t.Setenv("APP_ENV", "production")

	// (1) JWT_SECRET 없음 → 거부
	t.Setenv("JWT_SECRET", "")
	SecretFromEnv(nil)
	if !called || code != 1 {
		t.Fatalf("운영·미설정은 exit(1) 이어야: called=%v code=%d", called, code)
	}

	// (2) 개발용 기본값 그대로 → 거부
	called, code = false, 0
	t.Setenv("JWT_SECRET", devSecret)
	SecretFromEnv(nil)
	if !called || code != 1 {
		t.Fatalf("운영·기본값은 exit(1) 이어야: called=%v code=%d", called, code)
	}

	// (3) 제대로 된 값 → 통과(거부 안 함)
	called = false
	t.Setenv("JWT_SECRET", "prod-strong-secret-please-32bytes-xyz")
	if got := SecretFromEnv(nil); got != "prod-strong-secret-please-32bytes-xyz" || called {
		t.Fatalf("운영·정상값은 통과해야: got=%q called=%v", got, called)
	}
}
