package app

import (
	"context"
	"errors"

	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
)

// EventPublisher 는 회원 컨텍스트가 이벤트를 발행하는 포트.
type EventPublisher interface {
	Publish(ctx context.Context, events ...domain.DomainEvent) error
}

// CustomerService 는 회원 등록·조회 유스케이스.
type CustomerService struct {
	repo      domain.CustomerRepository
	publisher EventPublisher
}

func NewCustomerService(repo domain.CustomerRepository, publisher EventPublisher) *CustomerService {
	return &CustomerService{repo: repo, publisher: publisher}
}

// Register 는 회원을 등록한다. 이미 있으면 거부한다.
func (s *CustomerService) Register(ctx context.Context, id, email, name string) error {
	if _, err := s.repo.Find(ctx, domain.CustomerID(id)); err == nil {
		return domain.ErrCustomerExists
	} else if !errors.Is(err, domain.ErrCustomerNotFound) {
		return err
	}

	customer, err := domain.NewCustomer(domain.CustomerID(id), email, name)
	if err != nil {
		return err
	}
	if err := s.repo.Save(ctx, customer); err != nil {
		return err
	}
	return s.publisher.Publish(ctx, customer.PullEvents()...)
}

func (s *CustomerService) Get(ctx context.Context, id string) (*domain.Customer, error) {
	return s.repo.Find(ctx, domain.CustomerID(id))
}
