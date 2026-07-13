package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const secret = "test-secret-키"

func TestIssue_그리고_Verify_왕복(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token, err := Issue(secret, "cust-42", time.Hour, now)
	if err != nil {
		t.Fatalf("발급 실패: %v", err)
	}
	claims, err := Verify(secret, token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("검증 실패: %v", err)
	}
	if claims.Subject != "cust-42" {
		t.Errorf("sub = %q, 원했던 값 cust-42", claims.Subject)
	}
}

func TestVerify_만료된_토큰은_거부한다(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token, _ := Issue(secret, "cust-1", time.Hour, now)
	// 발급 2시간 뒤 검증 → 만료.
	if _, err := Verify(secret, token, now.Add(2*time.Hour)); err != ErrExpiredToken {
		t.Errorf("err = %v, 원했던 값 ErrExpiredToken", err)
	}
}

func TestVerify_다른_키로_서명검증하면_실패한다(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token, _ := Issue(secret, "cust-1", time.Hour, now)
	if _, err := Verify("다른-키", token, now); err != ErrBadSignature {
		t.Errorf("err = %v, 원했던 값 ErrBadSignature", err)
	}
}

func TestVerify_변조된_페이로드는_서명불일치(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	token, _ := Issue(secret, "cust-1", time.Hour, now)
	// payload 를 다른 회원으로 바꿔치기한 뒤 원래 서명을 붙이면 서명이 안 맞아야 한다.
	forged := Issue2(t, "cust-공격자", now) + tokenSig(token)
	if _, err := Verify(secret, forged, now); err == nil {
		t.Error("변조 토큰이 통과되면 안 된다")
	}
}

func TestMiddleware_토큰없으면_401_있으면_통과(t *testing.T) {
	now := time.Now()
	good, _ := Issue(secret, "cust-9", time.Hour, now)

	var seenSubject string
	protected := Middleware(secret)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenSubject = Subject(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// 1) 토큰 없음 → 401
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("토큰 없음: 상태 %d, 원했던 값 401", rec.Code)
	}

	// 2) 유효 토큰 → 통과 + 신원 주입
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+good)
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("유효 토큰: 상태 %d, 원했던 값 200", rec.Code)
	}
	if seenSubject != "cust-9" {
		t.Errorf("주입된 sub = %q, 원했던 값 cust-9", seenSubject)
	}
}

// --- 테스트 헬퍼: 변조 토큰 조립용 ---

// Issue2 는 헤더.페이로드(서명 앞 두 조각)까지만 만든다.
func Issue2(t *testing.T, subject string, now time.Time) string {
	t.Helper()
	full, err := Issue(secret, subject, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	i := lastDot(full)
	return full[:i+1]
}

func tokenSig(token string) string { return token[lastDot(token)+1:] }

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}
