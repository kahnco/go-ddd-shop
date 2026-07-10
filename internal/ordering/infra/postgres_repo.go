package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kahnco/go-ddd-shop/internal/ordering/domain"
	"github.com/kahnco/go-ddd-shop/internal/platform/telemetry"
)

// 주문 이벤트가 나가는 subject 접두사.
const outboxSubjectPrefix = "ordering"

// PostgresOrderRepository 는 OrderRepository 포트를 PostgreSQL 로 구현한 어댑터.
// 3편의 인메모리 구현을 이걸로 갈아끼우면, 여러 파드가 같은 상태를 공유한다.
// 도메인·애플리케이션 코드는 (또) 한 줄도 안 바뀐다 — 포트/어댑터의 마지막 배당금이다.
type PostgresOrderRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresOrderRepository 는 커넥션 풀을 만들고 스키마를 보장한다.
func NewPostgresOrderRepository(ctx context.Context, dsn string) (*PostgresOrderRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pg 풀 생성: %w", err)
	}
	repo := &PostgresOrderRepository{pool: pool}
	if err := repo.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return repo, nil
}

// ensureSchema 는 데모 편의를 위해 앱 기동 시 테이블을 만든다.
// 실서비스라면 golang-migrate·goose 같은 마이그레이션 도구로 버전 관리한다.
func (r *PostgresOrderRepository) ensureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS orders (
    id          TEXT PRIMARY KEY,
    customer_id TEXT NOT NULL,
    status      TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS order_lines (
    order_id   TEXT   NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id TEXT   NOT NULL,
    quantity   INT    NOT NULL,
    unit_price BIGINT NOT NULL
);
CREATE TABLE IF NOT EXISTS outbox (
    id             BIGSERIAL PRIMARY KEY,
    subject        TEXT   NOT NULL,
    event_name     TEXT   NOT NULL,
    payload        JSONB  NOT NULL,
    correlation_id TEXT,
    traceparent    TEXT,
    published_at   TIMESTAMPTZ
);`
	if _, err := r.pool.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("스키마 생성: %w", err)
	}
	return nil
}

// Ping 은 readiness probe 용 — DB 가 응답하는지 확인한다.
func (r *PostgresOrderRepository) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }
func (r *PostgresOrderRepository) Close()                         { r.pool.Close() }

// Save 는 주문과 항목을 한 트랜잭션으로 저장한다(upsert).
// 총액(total)은 저장하지 않는다 — 도메인이 항상 항목에서 계산하므로, DB 에도 파생값을
// 중복 저장하지 않아 "총액 = 소계 합" 불변식이 저장소에서도 깨질 여지가 없다.
func (r *PostgresOrderRepository) Save(ctx context.Context, order *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint: 커밋 성공 후의 Rollback 은 무해(no-op)

	_, err = tx.Exec(ctx, `
        INSERT INTO orders (id, customer_id, status) VALUES ($1, $2, $3)
        ON CONFLICT (id) DO UPDATE SET customer_id = EXCLUDED.customer_id, status = EXCLUDED.status`,
		string(order.ID()), string(order.CustomerID()), string(order.Status()))
	if err != nil {
		return err
	}

	// 항목은 통째로 지우고 다시 넣는다(애그리거트를 하나의 단위로 저장).
	if _, err = tx.Exec(ctx, `DELETE FROM order_lines WHERE order_id = $1`, string(order.ID())); err != nil {
		return err
	}
	for _, l := range order.Lines() {
		if _, err = tx.Exec(ctx, `
            INSERT INTO order_lines (order_id, product_id, quantity, unit_price) VALUES ($1, $2, $3, $4)`,
			string(order.ID()), string(l.ProductID()), l.Quantity().Value(), l.UnitPrice().Amount()); err != nil {
			return err
		}
	}

	// 핵심: 애그리거트가 낸 도메인 이벤트를 "같은 트랜잭션"으로 아웃박스에 적재한다.
	// 주문 저장과 이벤트 기록이 원자적이라, "저장은 됐는데 발행은 안 된" 불일치가 원천 차단된다.
	// 실제 브로커 발행은 별도의 릴레이(OutboxRelay)가 아웃박스를 읽어 대신 한다.
	cid := telemetry.CorrelationID(ctx)
	tp := telemetry.TraceparentFromContext(ctx) // 발행 시점에 소비자가 이어받을 trace 컨텍스트
	for _, e := range order.PullEvents() {
		payload, err := json.Marshal(e)
		if err != nil {
			return err
		}
		subject := outboxSubjectPrefix + "." + e.EventName()
		if _, err = tx.Exec(ctx, `
            INSERT INTO outbox (subject, event_name, payload, correlation_id, traceparent) VALUES ($1, $2, $3, $4, $5)`,
			subject, e.EventName(), payload, cid, tp); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// OutboxMessage 는 아웃박스에 적재된 미발행 이벤트 한 건.
type OutboxMessage struct {
	ID            int64
	Subject       string
	EventName     string
	Payload       []byte
	CorrelationID string
	Traceparent   string
}

// FetchOutbox 는 아직 발행하지 않은 이벤트를 오래된 순으로 가져온다(릴레이용).
func (r *PostgresOrderRepository) FetchOutbox(ctx context.Context, limit int) ([]OutboxMessage, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, subject, event_name, payload, COALESCE(correlation_id, ''), COALESCE(traceparent, '')
        FROM outbox WHERE published_at IS NULL ORDER BY id LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []OutboxMessage
	for rows.Next() {
		var m OutboxMessage
		if err := rows.Scan(&m.ID, &m.Subject, &m.EventName, &m.Payload, &m.CorrelationID, &m.Traceparent); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// MarkOutboxPublished 는 발행 완료한 이벤트에 발행 시각을 찍어 다시 보내지 않게 한다.
func (r *PostgresOrderRepository) MarkOutboxPublished(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx, `UPDATE outbox SET published_at = now() WHERE id = ANY($1)`, ids)
	return err
}

// FindByID 는 주문과 항목을 읽어 애그리거트로 되살린다(reconstitution).
func (r *PostgresOrderRepository) FindByID(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	var customerID, status string
	err := r.pool.QueryRow(ctx, `SELECT customer_id, status FROM orders WHERE id = $1`, string(id)).
		Scan(&customerID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.pool.Query(ctx, `
        SELECT product_id, quantity, unit_price FROM order_lines WHERE order_id = $1`, string(id))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []domain.OrderLine
	for rows.Next() {
		var productID string
		var quantity int
		var unitPrice int64
		if err := rows.Scan(&productID, &quantity, &unitPrice); err != nil {
			return nil, err
		}
		// 저장할 때 유효했던 값이므로 New* 는 실패하지 않는다.
		qty, err := domain.NewQuantity(quantity)
		if err != nil {
			return nil, err
		}
		price, err := domain.NewMoney(unitPrice)
		if err != nil {
			return nil, err
		}
		lines = append(lines, domain.NewOrderLine(domain.ProductID(productID), qty, price))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return domain.ReconstituteOrder(id, domain.CustomerID(customerID), lines, domain.OrderStatus(status)), nil
}
