package app

import (
	"context"

	"github.com/kahnco/go-ddd-shop/internal/catalog/domain"
)

// EventPublisher 는 카탈로그가 자신의 이벤트를 발행하는 포트.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}

// CatalogService 는 상품 등록·가격 변경·조회 유스케이스를 담는다.
type CatalogService struct {
	repo      domain.ProductRepository
	publisher EventPublisher
}

func NewCatalogService(repo domain.ProductRepository, publisher EventPublisher) *CatalogService {
	return &CatalogService{repo: repo, publisher: publisher}
}

// AddProduct 는 상품을 등록하고 ProductAdded 를 발행한다.
func (s *CatalogService) AddProduct(ctx context.Context, id, name string, price int64) error {
	product := domain.NewProduct(domain.ProductID(id), name, price)
	if err := s.repo.Save(ctx, product); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, product.PullEvents()...)
}

// ChangePrice 는 상품 가격을 바꾸고 ProductPriceChanged 를 발행한다.
func (s *CatalogService) ChangePrice(ctx context.Context, id string, price int64) error {
	product, err := s.repo.Find(ctx, domain.ProductID(id))
	if err != nil {
		return err
	}
	product.ChangePrice(price)
	if err := s.repo.Save(ctx, product); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, product.PullEvents()...)
}

func (s *CatalogService) Get(ctx context.Context, id string) (*domain.Product, error) {
	return s.repo.Find(ctx, domain.ProductID(id))
}

func (s *CatalogService) List(ctx context.Context) ([]*domain.Product, error) {
	return s.repo.All(ctx)
}
