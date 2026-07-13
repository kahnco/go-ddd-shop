package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const (
	claimsKey ctxKey = iota
	tokenKey
)

// Middleware 는 라우트를 감싸는 인증 게이트다. Authorization: Bearer <JWT> 를
// 검증하고, 통과하면 Claims 와 원본 토큰을 컨텍스트에 실어 다음 핸들러로 넘긴다.
// 실패(토큰 없음·서명 불일치·만료)하면 401 로 끊는다.
//
// mux 전체가 아니라 보호할 라우트에만 개별로 두른다(헬스체크·/metrics 는 열어 둬야 하므로).
//
//	mux.Handle("POST /orders", authMW(http.HandlerFunc(h.placeOrder)))
func Middleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r)
			if raw == "" {
				unauthorized(w, "인증 토큰이 필요합니다")
				return
			}
			claims, err := Verify(secret, raw, time.Now())
			if err != nil {
				unauthorized(w, err.Error())
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			ctx = context.WithValue(ctx, tokenKey, raw)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Subject 는 인증된 회원 ID 를 돌려준다. 미들웨어를 통과하지 않았으면 "".
func Subject(ctx context.Context) string {
	if c, ok := ctx.Value(claimsKey).(Claims); ok {
		return c.Subject
	}
	return ""
}

// Token 은 컨텍스트에 실린 원본 JWT 를 돌려준다.
// 하위 서비스를 호출할 때 그대로 전달해 신원을 이어 나른다(예: 장바구니→주문).
func Token(ctx context.Context) string {
	if t, ok := ctx.Value(tokenKey).(string); ok {
		return t
	}
	return ""
}

func bearerToken(r *http.Request) string {
	if after, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
