package domain

import "errors"

// 도메인 에러. 불변식 위반이나 규칙 위반을 도메인 언어로 표현한다.
// errors.Is 로 판별할 수 있게 센티넬 에러로 정의한다.
var (
	ErrEmptyOrder              = errors.New("주문에는 최소 하나의 항목이 필요합니다")
	ErrNegativeMoney           = errors.New("금액은 음수일 수 없습니다")
	ErrNonPositiveQuantity     = errors.New("수량은 1 이상이어야 합니다")
	ErrInvalidStatusTransition = errors.New("허용되지 않은 상태 전이")
	ErrOrderNotFound           = errors.New("주문을 찾을 수 없습니다")
	ErrUnknownProduct          = errors.New("카탈로그에 없는 상품입니다")
)
