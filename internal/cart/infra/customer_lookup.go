package infra

import (
	"context"
	"fmt"
	"net/http"
)

// HTTPCustomerLookup 은 CustomerLookup 포트를 회원 서비스 HTTP 호출로 구현한다.
// 동기 질의다 — 결제 순간 회원 존재를 즉시 확인해야 하므로.
type HTTPCustomerLookup struct {
	baseURL string
}

func NewHTTPCustomerLookup(baseURL string) *HTTPCustomerLookup {
	return &HTTPCustomerLookup{baseURL: baseURL}
}

func (l *HTTPCustomerLookup) Exists(ctx context.Context, customerID string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.baseURL+"/customers/"+customerID, nil)
	if err != nil {
		return false, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	default:
		return false, fmt.Errorf("회원 조회 실패: 상태 %d", resp.StatusCode)
	}
}
