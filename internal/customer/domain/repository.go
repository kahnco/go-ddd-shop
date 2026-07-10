package domain

import (
	"context"
	"errors"
)

var (
	ErrInvalidCustomer  = errors.New("이메일은 비어 있을 수 없습니다")
	ErrCustomerExists   = errors.New("이미 등록된 회원입니다")
	ErrCustomerNotFound = errors.New("회원을 찾을 수 없습니다")
)

// CustomerRepository 는 회원 저장소 포트.
type CustomerRepository interface {
	Save(ctx context.Context, customer *Customer) error
	Find(ctx context.Context, id CustomerID) (*Customer, error)
}
