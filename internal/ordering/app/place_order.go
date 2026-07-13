package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// PlaceOrderCommand 는 "주문하기" 유스케이스의 입력.
// 가격은 받지 않는다 — 클라이언트가 정하는 게 아니라 카탈로그가 정하기 때문이다.
type PlaceOrderCommand struct {
	CustomerID string
	Items      []OrderItemInput
	Channel    string // 주문 유입 경로(web/app/…). 비면 도메인에서 "web" 으로 기본값.
}

type OrderItemInput struct {
	ProductID string
	Quantity  int
}

// OrderService 는 주문 관련 유스케이스를 담는 애플리케이션 서비스.
// 도메인·인프라를 조율(orchestrate)할 뿐, 비즈니스 규칙은 도메인에 있다.
type OrderService struct {
	repo      domain.OrderRepository
	publisher EventPublisher
	ids       IDGenerator
	prices    ProductPriceLookup
}

func NewOrderService(repo domain.OrderRepository, publisher EventPublisher, ids IDGenerator, prices ProductPriceLookup) *OrderService {
	return &OrderService{repo: repo, publisher: publisher, ids: ids, prices: prices}
}

// PlaceOrder 유스케이스: 입력을 도메인으로 변환 → 애그리거트 생성 → 저장 → 이벤트 발행.
// 규칙 검증은 전부 도메인(값 객체·PlaceOrder)이 하고, 여기선 흐름만 엮는다.
func (s *OrderService) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (domain.OrderID, error) {
	lines := make([]domain.OrderLine, 0, len(cmd.Items))
	for _, item := range cmd.Items {
		qty, err := domain.NewQuantity(item.Quantity)
		if err != nil {
			return "", err
		}
		// 가격은 요청이 아니라 카탈로그(프로젝션)에서 가져온다 — 클라이언트 가격 조작을 막는다.
		price, err := s.prices.PriceOf(ctx, domain.ProductID(item.ProductID))
		if err != nil {
			return "", err
		}
		lines = append(lines, domain.NewOrderLine(domain.ProductID(item.ProductID), qty, price))
	}

	order, err := domain.PlaceOrder(s.ids.NewOrderID(), domain.CustomerID(cmd.CustomerID), lines, cmd.Channel)
	if err != nil {
		return "", err
	}

	if err := s.repo.Save(ctx, order); err != nil {
		return "", err
	}
	if err := s.publisher.Publish(ctx, order.PullEvents()...); err != nil {
		return "", err
	}
	return order.ID(), nil
}

// GetOrder 는 주문 조회 유스케이스.
func (s *OrderService) GetOrder(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	return s.repo.FindByID(ctx, id)
}
