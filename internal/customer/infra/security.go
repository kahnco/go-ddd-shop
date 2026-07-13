package infra

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kahnco/go-ddd-shop/internal/platform/auth"
)

// BcryptHasher 는 PasswordHasher 포트를 bcrypt 로 구현한다.
// bcrypt 는 솔트를 내장하고 비용(cost)을 조절할 수 있어 비밀번호 저장의 표준 선택이다.
type BcryptHasher struct{ cost int }

func NewBcryptHasher() BcryptHasher { return BcryptHasher{cost: bcrypt.DefaultCost} }

func (h BcryptHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	return string(b), err
}

func (h BcryptHasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// JWTIssuer 는 TokenIssuer 포트를 공유 auth 패키지의 HS256 JWT 로 구현한다.
type JWTIssuer struct {
	secret string
	ttl    time.Duration
}

func NewJWTIssuer(secret string, ttl time.Duration) JWTIssuer {
	return JWTIssuer{secret: secret, ttl: ttl}
}

func (i JWTIssuer) Issue(customerID string) (string, error) {
	return auth.Issue(i.secret, customerID, i.ttl, time.Now())
}

// RandomIDGenerator 는 IDGenerator 포트를 랜덤 16진 문자열로 구현한다("cust-" 접두).
type RandomIDGenerator struct{}

func (RandomIDGenerator) NewID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "cust-" + hex.EncodeToString(b[:])
}
