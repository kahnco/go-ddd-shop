package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/eventbus"
)

// ProductProjection 은 주문 컨텍스트가 들고 있는 "가격 읽기 모델"이다.
// 카탈로그의 이벤트(product.added·product.price_changed)를 받아 로컬 맵을 갱신하고,
// PlaceOrder 가 여기서 가격을 조회한다(ProductPriceLookup 포트 구현).
//
// 이게 CQRS 의 축소판이다 — 카탈로그가 쓰기 모델(source of truth)이라면,
// 이 프로젝션은 주문이 읽기에 최적화해 들고 있는 사본이다. 카탈로그를 매번 동기 호출하지
// 않으니 결합이 느슨하지만, 대신 가격이 잠깐 낡을 수 있다(결과적 일관성).
type ProductProjection struct {
	mu     sync.RWMutex
	prices map[domain.ProductID]int64
}

func NewProductProjection() *ProductProjection {
	return &ProductProjection{prices: make(map[domain.ProductID]int64)}
}

// PriceOf 는 ProductPriceLookup 포트 구현. 모르는 상품이면 ErrUnknownProduct.
func (p *ProductProjection) PriceOf(_ context.Context, id domain.ProductID) (domain.Money, error) {
	p.mu.RLock()
	price, ok := p.prices[id]
	p.mu.RUnlock()
	if !ok {
		return domain.Money{}, domain.ErrUnknownProduct
	}
	return domain.NewMoney(price)
}

func (p *ProductProjection) set(id domain.ProductID, price int64) {
	p.mu.Lock()
	p.prices[id] = price
	p.mu.Unlock()
}

// Empty 는 프로젝션이 비었는지 본다(부트스트랩 실패 시 기본 시드 여부 판단용).
func (p *ProductProjection) Empty() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.prices) == 0
}

// SeedDefault 는 카탈로그 없이 단독 실행할 때를 위한 기본 상품 시드.
func (p *ProductProjection) SeedDefault(id string, price int64) { p.set(domain.ProductID(id), price) }

// --- 카탈로그 이벤트 구독 핸들러 ---

func (p *ProductProjection) OnProductAdded(env eventbus.Envelope) error {
	var payload struct {
		ProductID string `json:"product_id"`
		Price     int64  `json:"price"`
	}
	if err := env.Into(&payload); err != nil {
		return err
	}
	p.set(domain.ProductID(payload.ProductID), payload.Price)
	return nil
}

func (p *ProductProjection) OnProductPriceChanged(env eventbus.Envelope) error {
	return p.OnProductAdded(env) // 모양이 같다(product_id + price)
}

// Bootstrap 은 시작 시 카탈로그의 현재 상태를 동기로 한 번 읽어 프로젝션을 채운다.
// (이벤트만으로는 이미 등록된 상품을 놓칠 수 있으니 — 콜드 스타트 대비.)
func (p *ProductProjection) Bootstrap(ctx context.Context, catalogBaseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catalogBaseURL+"/products", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("카탈로그 부트스트랩 실패: 상태 %d", resp.StatusCode)
	}

	var products []struct {
		ProductID string `json:"product_id"`
		Price     int64  `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return err
	}
	for _, pr := range products {
		p.set(domain.ProductID(pr.ProductID), pr.Price)
	}
	return nil
}
