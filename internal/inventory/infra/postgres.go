package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kahnco/go-ddd-shop/internal/inventory/domain"
)

// PostgresStore 는 재고·예약을 PostgreSQL 로 구현한 어댑터.
// 인메모리 구현과 달리 여러 inventory replica 가 같은 재고를 공유하며,
// 재고 수정은 행 잠금(SELECT … FOR UPDATE)으로 직렬화해 동시성 레이스를 원천 차단한다.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pg 풀 생성: %w", err)
	}
	s := &PostgresStore{pool: pool}
	if err := s.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS stock (
    product_id TEXT PRIMARY KEY,
    available  INT  NOT NULL
);
CREATE TABLE IF NOT EXISTS reservations (
    order_id   TEXT NOT NULL,
    product_id TEXT NOT NULL,
    quantity   INT  NOT NULL,
    PRIMARY KEY (order_id, product_id)
);`
	// 여러 replica 동시 기동 시 DDL 경합(pg_type 23505)을 피하려 자문 잠금으로 직렬화.
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(727275)`); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(727275)`)
	if _, err := conn.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("스키마 생성: %w", err)
	}
	return nil
}

func (s *PostgresStore) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
func (s *PostgresStore) Close()                         { s.pool.Close() }

// Seed 는 초기 재고를 채운다(이미 있으면 그대로).
func (s *PostgresStore) Seed(ctx context.Context, id domain.ProductID, available int) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO stock (product_id, available) VALUES ($1, $2) ON CONFLICT (product_id) DO NOTHING`,
		string(id), available)
	return err
}

// --- StockRepository ---

func (s *PostgresStore) FindByProduct(ctx context.Context, id domain.ProductID) (*domain.StockItem, error) {
	var available int
	err := s.pool.QueryRow(ctx, `SELECT available FROM stock WHERE product_id = $1`, string(id)).Scan(&available)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrStockItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return domain.NewStockItem(id, available), nil
}

// Update 는 재고 행을 FOR UPDATE 로 잠근 채 조회·수정·저장한다.
// 같은 상품에 동시 요청이 와도 행 잠금으로 직렬화돼, read-modify-write 레이스가 없다.
func (s *PostgresStore) Update(ctx context.Context, id domain.ProductID, mutate func(*domain.StockItem) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var available int
	err = tx.QueryRow(ctx, `SELECT available FROM stock WHERE product_id = $1 FOR UPDATE`, string(id)).Scan(&available)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrStockItemNotFound
	}
	if err != nil {
		return err
	}

	item := domain.NewStockItem(id, available)
	if err := mutate(item); err != nil {
		return err // 비즈니스 오류(재고 부족 등) → 롤백
	}
	if _, err := tx.Exec(ctx, `UPDATE stock SET available = $1 WHERE product_id = $2`, item.Available(), string(id)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// --- ReservationRepository ---

func (s *PostgresStore) Save(ctx context.Context, res *domain.Reservation) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM reservations WHERE order_id = $1`, string(res.OrderID())); err != nil {
		return err
	}
	for _, item := range res.Items() {
		if _, err := tx.Exec(ctx,
			`INSERT INTO reservations (order_id, product_id, quantity) VALUES ($1, $2, $3)`,
			string(res.OrderID()), string(item.ProductID), item.Quantity); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *PostgresStore) Find(ctx context.Context, orderID domain.OrderID) (*domain.Reservation, error) {
	rows, err := s.pool.Query(ctx, `SELECT product_id, quantity FROM reservations WHERE order_id = $1`, string(orderID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := domain.NewReservation(orderID)
	found := false
	for rows.Next() {
		var productID string
		var qty int
		if err := rows.Scan(&productID, &qty); err != nil {
			return nil, err
		}
		res.Add(domain.ProductID(productID), qty)
		found = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if !found {
		return nil, domain.ErrReservationNotFound
	}
	return res, nil
}

func (s *PostgresStore) Delete(ctx context.Context, orderID domain.OrderID) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM reservations WHERE order_id = $1`, string(orderID))
	return err
}
