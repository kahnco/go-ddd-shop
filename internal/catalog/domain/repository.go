package domain

import (
	"context"
	"errors"
)

var ErrProductNotFound = errors.New("상품을 찾을 수 없습니다")

// ProductRepository 는 카탈로그 저장소 포트.
type ProductRepository interface {
	Save(ctx context.Context, product *Product) error
	Find(ctx context.Context, id ProductID) (*Product, error)
	All(ctx context.Context) ([]*Product, error)
}
