package auth

import (
	"log/slog"
	"os"
)

// devSecret 은 JWT_SECRET 이 없을 때 쓰는 개발용 기본값.
// 모든 서비스가 같은 기본값을 쓰므로 단독/로컬 실행에서도 토큰이 서로 통한다.
// 운영에서는 반드시 JWT_SECRET 을 주입해야 한다.
const devSecret = "dev-secret-change-me"

// exit 는 테스트에서 가로채기 위한 훅(기본은 os.Exit).
var exit = os.Exit

// SecretFromEnv 는 JWT_SECRET 을 읽는다.
//
// 심층 방어(defense in depth): 시크릿을 암호화해 주입하기로 했더라도, 누군가 그 주입을
// 깜빡할 수 있다. 그래서 APP_ENV=production 인데 JWT_SECRET 이 없거나 개발용 기본값이면
// 부팅을 거부한다 — 약한 키로 조용히 도느니, 시끄럽게 멈추는 게 낫다.
// 운영이 아니면(로컬·테스트) 개발용 기본값으로 떨어지고 경고만 남긴다.
func SecretFromEnv(logger *slog.Logger) string {
	s := os.Getenv("JWT_SECRET")
	insecure := s == "" || s == devSecret

	if os.Getenv("APP_ENV") == "production" && insecure {
		if logger != nil {
			logger.Error("운영(APP_ENV=production)에서 JWT_SECRET 이 없거나 기본값입니다 — 안전을 위해 부팅을 거부합니다")
		}
		exit(1)
		return "" // exit 가 훅으로 대체된 테스트에서만 도달
	}

	if insecure {
		if logger != nil {
			logger.Warn("JWT_SECRET 이 없어 개발용 기본 비밀키를 사용합니다 — 운영에서는 반드시 설정하세요")
		}
		return devSecret
	}
	return s
}
