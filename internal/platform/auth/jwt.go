// Package auth 는 서비스들이 공유하는 인증 조각이다.
// 외부 JWT 라이브러리 없이, 표준 라이브러리만으로 HS256(HMAC-SHA256) JWT 를
// 발급·검증한다. JWT 는 결국 base64url(header).base64url(payload).base64url(signature)
// 세 조각일 뿐이고, 서명이 HMAC 이면 이 정도로 충분하다.
//
// 회원 서비스가 로그인 시 Issue 로 토큰을 발급하고,
// 주문·장바구니 서비스가 Middleware 안에서 Verify 로 검증한다. 비밀키(secret)는
// 모든 서비스가 같은 값을 공유한다(대칭 키). 비대칭이 필요하면 RS256 으로 확장하면 된다.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptySecret    = errors.New("JWT 비밀키가 비어 있습니다")
	ErrMalformedToken = errors.New("토큰 형식이 올바르지 않습니다")
	ErrBadSignature   = errors.New("토큰 서명이 유효하지 않습니다")
	ErrExpiredToken   = errors.New("토큰이 만료되었습니다")
)

// Claims 는 토큰에 담는 최소 정보. sub=회원 ID, iat=발급시각, exp=만료시각(유닉스 초).
type Claims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// base64url(패딩 없음) — JWT 규격이 요구하는 인코딩.
var b64 = base64.RawURLEncoding

// 고정 헤더({"alg":"HS256","typ":"JWT"}). 우리는 HS256 만 쓰므로 미리 인코딩해 둔다.
const encodedHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// Issue 는 subject(회원 ID)에 대해 now 기준 ttl 동안 유효한 HS256 JWT 를 만든다.
func Issue(secret, subject string, ttl time.Duration, now time.Time) (string, error) {
	if secret == "" {
		return "", ErrEmptySecret
	}
	claims := Claims{Subject: subject, IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix()}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := encodedHeader + "." + b64.EncodeToString(payloadJSON)
	return signingInput + "." + sign(secret, signingInput), nil
}

// Verify 는 서명과 만료를 검증하고 Claims 를 돌려준다.
func Verify(secret, token string, now time.Time) (Claims, error) {
	if secret == "" {
		return Claims{}, ErrEmptySecret
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrMalformedToken
	}
	signingInput := parts[0] + "." + parts[1]
	// 상수 시간 비교로 타이밍 공격을 막는다.
	if subtle.ConstantTimeCompare([]byte(sign(secret, signingInput)), []byte(parts[2])) != 1 {
		return Claims{}, ErrBadSignature
	}
	payloadJSON, err := b64.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrMalformedToken
	}
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return Claims{}, ErrMalformedToken
	}
	if now.Unix() >= claims.ExpiresAt {
		return Claims{}, ErrExpiredToken
	}
	return claims, nil
}

func sign(secret, input string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(input))
	return b64.EncodeToString(mac.Sum(nil))
}
