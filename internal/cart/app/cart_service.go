package app

import (
	"context"
	"errors"

	"github.com/kahnco/go-ddd-shop/internal/cart/domain"
)

// 장바구니가 바깥 컨텍스트에 요구하는 포트들.
// EDD 가 아니라 "동기 질의/명령"으로 잇는 게 자연스러운 경우다 —
// 결제는 즉시 회원을 확인하고 즉시 주문 번호를 받아야 하므로.

// CustomerLookup 은 회원이 존재하는지 확인하는 포트(회원 서비스로).
type CustomerLookup interface {
	Exists(ctx context.Context, customerID string) (bool, error)
}

// OrderPlacer 는 장바구니 내용으로 주문을 생성하는 포트(주문 서비스로).
type OrderPlacer interface {
	Place(ctx context.Context, customerID string, items []OrderItem) (orderID string, err error)
}

type OrderItem struct {
	ProductID string
	Quantity  int
}

// ErrCustomerNotRegistered 는 등록되지 않은 회원이 결제를 시도할 때.
var ErrCustomerNotRegistered = errors.New("등록된 회원만 결제할 수 있습니다")

// CartService 는 장바구니 담기·빼기·조회·결제 유스케이스.
type CartService struct {
	carts     domain.CartRepository
	customers CustomerLookup
	orders    OrderPlacer
}

func NewCartService(carts domain.CartRepository, customers CustomerLookup, orders OrderPlacer) *CartService {
	return &CartService{carts: carts, customers: customers, orders: orders}
}

func (s *CartService) AddItem(ctx context.Context, customerID, productID string, quantity int) (*domain.Cart, error) {
	cart, err := s.load(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if err := cart.AddItem(productID, quantity); err != nil {
		return nil, err
	}
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) RemoveItem(ctx context.Context, customerID, productID string) (*domain.Cart, error) {
	cart, err := s.load(ctx, customerID)
	if err != nil {
		return nil, err
	}
	cart.RemoveItem(productID)
	if err := s.carts.Save(ctx, cart); err != nil {
		return nil, err
	}
	return cart, nil
}

func (s *CartService) Get(ctx context.Context, customerID string) (*domain.Cart, error) {
	return s.load(ctx, customerID)
}

// Checkout 은 장바구니를 주문으로 바꾼다.
// 1) 회원인지 확인 → 2) 빈 장바구니 거부 → 3) 주문 생성 → 4) 장바구니 비우기.
func (s *CartService) Checkout(ctx context.Context, customerID string) (string, error) {
	ok, err := s.customers.Exists(ctx, customerID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ErrCustomerNotRegistered
	}

	cart, err := s.load(ctx, customerID)
	if err != nil {
		return "", err
	}
	if cart.IsEmpty() {
		return "", domain.ErrEmptyCart
	}

	items := make([]OrderItem, 0)
	for _, it := range cart.Items() {
		items = append(items, OrderItem{ProductID: it.ProductID, Quantity: it.Quantity})
	}
	orderID, err := s.orders.Place(ctx, customerID, items)
	if err != nil {
		return "", err
	}

	// 주문이 만들어졌으니 장바구니를 비운다.
	if err := s.carts.Delete(ctx, customerID); err != nil {
		return "", err
	}
	return orderID, nil
}

// load 는 기존 장바구니를 가져오거나, 없으면 빈 장바구니를 만든다.
func (s *CartService) load(ctx context.Context, customerID string) (*domain.Cart, error) {
	cart, err := s.carts.Find(ctx, customerID)
	if errors.Is(err, domain.ErrCartNotFound) {
		return domain.NewCart(customerID), nil
	}
	if err != nil {
		return nil, err
	}
	return cart, nil
}
