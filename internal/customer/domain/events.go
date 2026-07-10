package domain

type DomainEvent interface {
	EventName() string
}

// CustomerRegistered — 회원 등록됨. (예: 환영 메일·마케팅 컨텍스트가 구독할 수 있다.)
type CustomerRegistered struct {
	CustomerID CustomerID `json:"customer_id"`
	Email      string     `json:"email"`
}

func (CustomerRegistered) EventName() string { return "customer.registered" }
