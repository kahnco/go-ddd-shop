package auth

import (
	"log/slog"
	"os"
)

// devSecret 은 JWT_SECRET 이 없을 때 쓰는 개발용 기본값.
// 모든 서비스가 같은 기본값을 쓰므로 단독/로컬 실행에서도 토큰이 서로 통한다.
// 운영에서는 반드시 JWT_SECRET 을 주입해야 한다(경고 로그로 알린다).
const devSecret = "dev-secret-change-me"

// SecretFromEnv 는 JWT_SECRET 을 읽는다. 없으면 개발용 기본값을 쓰고 경고한다.
func SecretFromEnv(logger *slog.Logger) string {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return s
	}
	if logger != nil {
		logger.Warn("JWT_SECRET 이 없어 개발용 기본 비밀키를 사용합니다 — 운영에서는 반드시 설정하세요")
	}
	return devSecret
}
