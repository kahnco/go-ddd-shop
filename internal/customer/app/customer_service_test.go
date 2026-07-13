package app

import (
	"context"
	"errors"
	"testing"

	"github.com/kahnco/go-ddd-shop/internal/customer/domain"
)

type fakeRepo struct {
	store map[domain.CustomerID]*domain.Customer
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[domain.CustomerID]*domain.Customer{}} }
func (r *fakeRepo) Save(_ context.Context, c *domain.Customer) error {
	r.store[c.ID()] = c
	return nil
}
func (r *fakeRepo) Find(_ context.Context, id domain.CustomerID) (*domain.Customer, error) {
	c, ok := r.store[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return c, nil
}
func (r *fakeRepo) FindByEmail(_ context.Context, email string) (*domain.Customer, error) {
	for _, c := range r.store {
		if c.Email() == email {
			return c, nil
		}
	}
	return nil, domain.ErrCustomerNotFound
}

type fakePublisher struct{ published []domain.DomainEvent }

func (p *fakePublisher) Publish(_ context.Context, e ...domain.DomainEvent) error {
	p.published = append(p.published, e...)
	return nil
}

// 테스트용 가짜 어댑터들: 해시는 "hashed:<plain>", 토큰은 "tok:<id>", ID 는 순번.
type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }
func (fakeHasher) Compare(hash, plain string) bool   { return hash == "hashed:"+plain }

type fakeIssuer struct{}

func (fakeIssuer) Issue(customerID string) (string, error) { return "tok:" + customerID, nil }

type seqIDs struct{ n int }

func (g *seqIDs) NewID() string { g.n++; return "cust-" + string(rune('0'+g.n)) }

func newSvc() (*CustomerService, *fakePublisher) {
	pub := &fakePublisher{}
	return NewCustomerService(newFakeRepo(), pub, fakeHasher{}, fakeIssuer{}, &seqIDs{}), pub
}

func TestRegister_해시저장_이벤트발행(t *testing.T) {
	svc, pub := newSvc()

	id, err := svc.Register(context.Background(), "a@b.com", "supersecret", "홍길동")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if _, ok := pub.published[0].(domain.CustomerRegistered); !ok {
		t.Fatalf("CustomerRegistered 발행돼야 함: %T", pub.published[0])
	}
	got, _ := svc.Get(context.Background(), string(id))
	if got.Email() != "a@b.com" {
		t.Fatalf("이메일 = a@b.com 여야 하는데 %s", got.Email())
	}
	if got.PasswordHash() != "hashed:supersecret" {
		t.Fatalf("비밀번호는 해시돼 저장돼야 함: %s", got.PasswordHash())
	}
}

func TestRegister_약한_비밀번호는_거부(t *testing.T) {
	svc, _ := newSvc()
	if _, err := svc.Register(context.Background(), "a@b.com", "short", "홍길동"); !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("약한 비밀번호는 ErrWeakPassword 여야 하는데: %v", err)
	}
}

func TestRegister_이메일_중복은_거부(t *testing.T) {
	svc, _ := newSvc()
	_, _ = svc.Register(context.Background(), "a@b.com", "supersecret", "홍길동")
	if _, err := svc.Register(context.Background(), "a@b.com", "anothersecret", "임꺽정"); !errors.Is(err, domain.ErrCustomerExists) {
		t.Fatalf("중복 이메일은 ErrCustomerExists 여야 하는데: %v", err)
	}
}

func TestLogin_성공하면_토큰발급(t *testing.T) {
	svc, _ := newSvc()
	id, _ := svc.Register(context.Background(), "a@b.com", "supersecret", "홍길동")

	token, err := svc.Login(context.Background(), "a@b.com", "supersecret")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token != "tok:"+string(id) {
		t.Fatalf("토큰 = tok:%s 여야 하는데 %s", id, token)
	}
}

func TestLogin_틀린_비밀번호나_없는_이메일은_같은_오류(t *testing.T) {
	svc, _ := newSvc()
	_, _ = svc.Register(context.Background(), "a@b.com", "supersecret", "홍길동")

	// 비밀번호 틀림, 그리고 없는 이메일 — 둘 다 ErrInvalidCredentials 여야 한다(계정 존재 은닉).
	if _, err := svc.Login(context.Background(), "a@b.com", "wrongpass"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("틀린 비밀번호: %v", err)
	}
	if _, err := svc.Login(context.Background(), "nobody@b.com", "whatever"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("없는 이메일: %v", err)
	}
}
