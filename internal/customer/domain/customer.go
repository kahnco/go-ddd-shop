// Package domain 은 회원(customer) bounded context 의 도메인이다.
package domain

type CustomerID string

// Customer 는 회원. 식별자·이메일·이름과 함께 비밀번호 해시를 가진다.
// 도메인은 해시 문자열만 들고 있을 뿐, 해싱·검증 방법(bcrypt 등)은 모른다 — 그건 인프라의 몫.
type Customer struct {
	id           CustomerID
	email        string
	name         string
	passwordHash string
	events       []DomainEvent
}

// NewCustomer 는 회원을 등록한다. 이메일이 비면 거부한다.
// passwordHash 는 이미 해시된 값을 받는다(평문은 도메인에 들어오지 않는다).
func NewCustomer(id CustomerID, email, name, passwordHash string) (*Customer, error) {
	if email == "" {
		return nil, ErrInvalidCustomer
	}
	c := &Customer{id: id, email: email, name: name, passwordHash: passwordHash}
	c.record(CustomerRegistered{CustomerID: id, Email: email})
	return c, nil
}

func (c *Customer) ID() CustomerID       { return c.id }
func (c *Customer) Email() string        { return c.email }
func (c *Customer) Name() string         { return c.name }
func (c *Customer) PasswordHash() string { return c.passwordHash }
func (c *Customer) record(e DomainEvent) { c.events = append(c.events, e) }
func (c *Customer) PullEvents() []DomainEvent {
	e := c.events
	c.events = nil
	return e
}
