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

// PasswordHasher 는 비밀번호를 해시하고 검증하는 포트(인프라: bcrypt).
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) bool
}

// TokenIssuer 는 로그인 성공 시 회원 ID 로 접근 토큰(JWT)을 발급하는 포트.
type TokenIssuer interface {
	Issue(customerID string) (string, error)
}

// IDGenerator 는 새 회원 ID 를 만든다(서버가 정한다 — 클라이언트가 못 정한다).
type IDGenerator interface {
	NewID() string
}

// CustomerService 는 회원 등록·로그인·조회 유스케이스.
type CustomerService struct {
	repo      domain.CustomerRepository
	publisher EventPublisher
	hasher    PasswordHasher
	tokens    TokenIssuer
	ids       IDGenerator
}

func NewCustomerService(
	repo domain.CustomerRepository,
	publisher EventPublisher,
	hasher PasswordHasher,
	tokens TokenIssuer,
	ids IDGenerator,
) *CustomerService {
	return &CustomerService{repo: repo, publisher: publisher, hasher: hasher, tokens: tokens, ids: ids}
}

// Register 는 회원을 등록한다. 이메일 중복·약한 비밀번호는 거부하고,
// 비밀번호는 해시해 저장한다. 회원 ID 는 서버가 생성한다.
func (s *CustomerService) Register(ctx context.Context, email, password, name string) (domain.CustomerID, error) {
	if len(password) < 8 {
		return "", domain.ErrWeakPassword
	}
	if _, err := s.repo.FindByEmail(ctx, email); err == nil {
		return "", domain.ErrCustomerExists
	} else if !errors.Is(err, domain.ErrCustomerNotFound) {
		return "", err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return "", err
	}
	id := domain.CustomerID(s.ids.NewID())
	customer, err := domain.NewCustomer(id, email, name, hash)
	if err != nil {
		return "", err
	}
	if err := s.repo.Save(ctx, customer); err != nil {
		return "", err
	}
	if err := s.publisher.Publish(ctx, customer.PullEvents()...); err != nil {
		return "", err
	}
	return id, nil
}

// Login 은 이메일·비밀번호를 검증하고 접근 토큰을 발급한다.
// 이메일이 없든 비밀번호가 틀리든 같은 오류(ErrInvalidCredentials)를 돌려준다 —
// "그 이메일은 없다"고 알려 주면 계정 존재 여부가 새기 때문이다.
func (s *CustomerService) Login(ctx context.Context, email, password string) (string, error) {
	customer, err := s.repo.FindByEmail(ctx, email)
	if errors.Is(err, domain.ErrCustomerNotFound) {
		return "", domain.ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if !s.hasher.Compare(customer.PasswordHash(), password) {
		return "", domain.ErrInvalidCredentials
	}
	return s.tokens.Issue(string(customer.ID()))
}

func (s *CustomerService) Get(ctx context.Context, id string) (*domain.Customer, error) {
	return s.repo.Find(ctx, domain.CustomerID(id))
}
