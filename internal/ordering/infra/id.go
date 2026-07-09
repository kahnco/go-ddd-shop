package infra

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// RandomIDGenerator 는 app.IDGenerator 포트를 crypto/rand 로 구현한 어댑터.
// 충돌 걱정이 적은 랜덤 식별자를 만든다.
type RandomIDGenerator struct{}

func (RandomIDGenerator) NewOrderID() domain.OrderID {
	b := make([]byte, 12)
	_, _ = rand.Read(b) // crypto/rand.Read 는 실패하지 않는다(문서상)
	return domain.OrderID("order_" + hex.EncodeToString(b))
}
