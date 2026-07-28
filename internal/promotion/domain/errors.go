package domain

import "errors"

var (
	// ErrEventNotFound — 존재하지 않는 이벤트에 응모.
	ErrEventNotFound = errors.New("promotion: 이벤트를 찾을 수 없음")
	// ErrNotStarted — 아직 시작 시각(StartsAt) 이전이라 응모 무효.
	ErrNotStarted = errors.New("promotion: 아직 시작 전인 이벤트")
)
