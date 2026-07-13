package domain

import (
	"context"
	"errors"
)

var (
	ErrInvalidCustomer    = errors.New("이메일은 비어 있을 수 없습니다")
	ErrCustomerExists     = errors.New("이미 등록된 회원입니다")
	ErrCustomerNotFound   = errors.New("회원을 찾을 수 없습니다")
	ErrWeakPassword       = errors.New("비밀번호는 8자 이상이어야 합니다")
	ErrInvalidCredentials = errors.New("이메일 또는 비밀번호가 올바르지 않습니다")
)

// CustomerRepository 는 회원 저장소 포트.
type CustomerRepository interface {
	Save(ctx context.Context, customer *Customer) error
	Find(ctx context.Context, id CustomerID) (*Customer, error)
	// FindByEmail 은 로그인 시 이메일로 회원을 찾는다. 없으면 ErrCustomerNotFound.
	FindByEmail(ctx context.Context, email string) (*Customer, error)
}
