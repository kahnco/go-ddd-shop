package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
)

// PlaceOrderCommand 는 "주문하기" 유스케이스의 입력.
// 애플리케이션 계층의 입력은 원시 타입(string·int)으로 받고,
// 도메인 값 객체로의 변환·검증은 유스케이스 안에서 한다.
type PlaceOrderCommand struct {
	CustomerID string
	Items      []OrderItemInput
}

type OrderItemInput struct {
	ProductID string
	Quantity  int
	UnitPrice int64
}

// OrderService 는 주문 관련 유스케이스를 담는 애플리케이션 서비스.
// 도메인·인프라를 조율(orchestrate)할 뿐, 비즈니스 규칙은 도메인에 있다.
type OrderService struct {
	repo      domain.OrderRepository
	publisher EventPublisher
	ids       IDGenerator
}

func NewOrderService(repo domain.OrderRepository, publisher EventPublisher, ids IDGenerator) *OrderService {
	return &OrderService{repo: repo, publisher: publisher, ids: ids}
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
		price, err := domain.NewMoney(item.UnitPrice)
		if err != nil {
			return "", err
		}
		lines = append(lines, domain.NewOrderLine(domain.ProductID(item.ProductID), qty, price))
	}

	order, err := domain.PlaceOrder(s.ids.NewOrderID(), domain.CustomerID(cmd.CustomerID), lines)
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
