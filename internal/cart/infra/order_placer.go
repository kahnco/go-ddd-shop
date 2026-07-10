package infra

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kahnco/go-ddd-shop/internal/cart/app"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// HTTPOrderPlacer 는 OrderPlacer 포트를 주문 서비스 HTTP 호출로 구현한다.
// 장바구니 항목을 주문 서비스가 이해하는 JSON 으로 번역한다(ACL). 가격은 싣지 않는다 —
// 주문 서비스가 카탈로그에서 정하기 때문이다.
type HTTPOrderPlacer struct {
	baseURL string
}

func NewHTTPOrderPlacer(baseURL string) *HTTPOrderPlacer {
	return &HTTPOrderPlacer{baseURL: baseURL}
}

func (p *HTTPOrderPlacer) Place(ctx context.Context, customerID string, items []app.OrderItem) (string, error) {
	type item struct {
		ProductID string `json:"product_id"`
		Quantity  int    `json:"quantity"`
	}
	reqBody := struct {
		CustomerID string `json:"customer_id"`
		Items      []item `json:"items"`
	}{CustomerID: customerID}
	for _, it := range items {
		reqBody.Items = append(reqBody.Items, item{ProductID: it.ProductID, Quantity: it.Quantity})
	}

	raw, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/orders", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	// 상관 ID 를 이어 전달해, 장바구니→주문→사가 전체를 한 ID 로 추적한다.
	if cid := telemetry.CorrelationID(ctx); cid != "" {
		req.Header.Set(telemetry.HeaderCorrelationID, cid)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var body struct {
		OrderID string `json:"order_id"`
		Error   string `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("주문 생성 실패(상태 %d): %s", resp.StatusCode, body.Error)
	}
	return body.OrderID, nil
}
