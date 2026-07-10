// Package domain 은 회원(customer) bounded context 의 도메인이다.
package domain

type CustomerID string

// Customer 는 회원. 식별자·이메일·이름을 가진다.
type Customer struct {
	id     CustomerID
	email  string
	name   string
	events []DomainEvent
}

// NewCustomer 는 회원을 등록한다. 이메일이 비면 거부한다.
func NewCustomer(id CustomerID, email, name string) (*Customer, error) {
	if email == "" {
		return nil, ErrInvalidCustomer
	}
	c := &Customer{id: id, email: email, name: name}
	c.record(CustomerRegistered{CustomerID: id, Email: email})
	return c, nil
}

func (c *Customer) ID() CustomerID       { return c.id }
func (c *Customer) Email() string        { return c.email }
func (c *Customer) Name() string         { return c.name }
func (c *Customer) record(e DomainEvent) { c.events = append(c.events, e) }
func (c *Customer) PullEvents() []DomainEvent {
	e := c.events
	c.events = nil
	return e
}
