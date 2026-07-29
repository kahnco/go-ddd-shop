package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kahnco/go-ddd-shop/internal/promotion/app"
	"github.com/kahnco/go-ddd-shop/internal/promotion/domain"
)

// PostgresRepo 는 응모를 PostgreSQL 로 기록하는 어댑터다.
// 카운터 행을 SELECT … FOR UPDATE 로 잠근 채 순번을 배정하므로, 여러 promotion replica 가
// 동시에 응모를 받아도 순번이 빈틈없이(gapless) 직렬화된다.
// 시퀀스(nextval)와 달리 트랜잭션이 롤백돼도 번호가 새지 않는다 — "정확히 N번째"에 필수다.
type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(ctx context.Context, dsn string) (*PostgresRepo, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pg 풀 생성: %w", err)
	}
	r := &PostgresRepo{pool: pool}
	if err := r.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return r, nil
}

func (r *PostgresRepo) ensureSchema(ctx context.Context) error {
	const ddl = `
CREATE TABLE IF NOT EXISTS promotion_event (
    event_id       TEXT PRIMARY KEY,
    target_seq     INT  NOT NULL,
    starts_at      TIMESTAMPTZ NOT NULL,
    winner_user_id TEXT
);
CREATE TABLE IF NOT EXISTS promotion_counter (
    event_id TEXT PRIMARY KEY,
    cnt      INT  NOT NULL
);
CREATE TABLE IF NOT EXISTS promotion_entry (
    event_id   TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    seq        INT  NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (event_id, user_id),
    UNIQUE (event_id, seq)
);
CREATE TABLE IF NOT EXISTS promotion_outbox (
    id           BIGSERIAL PRIMARY KEY,
    subject      TEXT NOT NULL,
    event_name   TEXT NOT NULL,
    payload      BYTEA NOT NULL,
    dedup_id     TEXT UNIQUE,
    published_at TIMESTAMPTZ
);`
	// 여러 replica 동시 기동 시 DDL 경합을 피하려 자문 잠금으로 직렬화(다른 컨텍스트와 같은 패턴).
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(919191)`); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(919191)`)
	if _, err := conn.Exec(ctx, ddl); err != nil {
		return fmt.Errorf("스키마 생성: %w", err)
	}
	return nil
}

func (r *PostgresRepo) Close()                          { r.pool.Close() }
func (r *PostgresRepo) Ping(ctx context.Context) error  { return r.pool.Ping(ctx) }

func (r *PostgresRepo) SeedEvent(ctx context.Context, e domain.Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO promotion_event (event_id, target_seq, starts_at) VALUES ($1,$2,$3)
		 ON CONFLICT (event_id) DO NOTHING`, e.ID, e.TargetSeq, e.StartsAt); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO promotion_counter (event_id, cnt) VALUES ($1, 0)
		 ON CONFLICT (event_id) DO NOTHING`, e.ID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Enter 는 카운터 행을 잠근 채 순번을 배정한다. 핵심은 세 가지다:
//  1. 시작 전·이미 응모는 카운터를 잠그기 전에 걸러 순번을 소비하지 않는다.
//  2. 카운터 전진(UPDATE)은 응모 INSERT 가 성공한 뒤에만 한다 → 롤백돼도 빈틈이 없다.
//  3. 모든 응모가 같은 카운터 행 위에서 FOR UPDATE 로 직렬화된다 → 정확히 한 명이 N번째.
func (r *PostgresRepo) Enter(ctx context.Context, eventID, userID string, now time.Time) (app.Result, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return app.Result{}, err
	}
	defer tx.Rollback(ctx)

	// 이벤트 로드(당첨 순번·시작 시각).
	var target int
	var startsAt time.Time
	err = tx.QueryRow(ctx,
		`SELECT target_seq, starts_at FROM promotion_event WHERE event_id=$1`, eventID).
		Scan(&target, &startsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Result{}, domain.ErrEventNotFound
	}
	if err != nil {
		return app.Result{}, err
	}
	if now.Before(startsAt) {
		return app.Result{}, domain.ErrNotStarted // 롤백 → 순번 미소비
	}

	// 멱등: 이미 응모했으면 기존 순번(순번 미소비).
	var seq int
	err = tx.QueryRow(ctx,
		`SELECT seq FROM promotion_entry WHERE event_id=$1 AND user_id=$2`, eventID, userID).Scan(&seq)
	if err == nil {
		return app.Result{Seq: seq, Winner: seq == target, Already: true}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return app.Result{}, err
	}

	// 카운터 행을 잠근다 — 여기서부터 이 이벤트의 응모는 한 줄로 직렬화된다.
	var cnt int
	if err := tx.QueryRow(ctx,
		`SELECT cnt FROM promotion_counter WHERE event_id=$1 FOR UPDATE`, eventID).Scan(&cnt); err != nil {
		return app.Result{}, err
	}
	next := cnt + 1

	// 응모 기록. (event,user) 유니크라, 같은 사용자가 락 밖에서 먼저 들어왔다면 충돌한다.
	ct, err := tx.Exec(ctx,
		`INSERT INTO promotion_entry (event_id, user_id, seq, created_at) VALUES ($1,$2,$3,$4)
		 ON CONFLICT (event_id, user_id) DO NOTHING`, eventID, userID, next, now)
	if err != nil {
		return app.Result{}, err
	}
	if ct.RowsAffected() == 0 {
		// 같은 사용자가 사이에 먼저 들어왔다 → 카운터를 전진시키지 않고 기존 순번 반환(빈틈 방지).
		if err := tx.QueryRow(ctx,
			`SELECT seq FROM promotion_entry WHERE event_id=$1 AND user_id=$2`, eventID, userID).Scan(&seq); err != nil {
			return app.Result{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return app.Result{}, err
		}
		return app.Result{Seq: seq, Winner: seq == target, Already: true}, nil
	}

	// 카운터 전진 — 성공한 응모에 대해서만. 롤백되면 이 증가도 사라져 빈틈이 없다.
	if _, err := tx.Exec(ctx,
		`UPDATE promotion_counter SET cnt=$1 WHERE event_id=$2`, next, eventID); err != nil {
		return app.Result{}, err
	}

	winner := next == target
	if winner {
		// 당첨 확정을 이벤트 행에 못박는다(감사 가능·정확히 하나).
		if _, err := tx.Exec(ctx,
			`UPDATE promotion_event SET winner_user_id=$1 WHERE event_id=$2 AND winner_user_id IS NULL`,
			userID, eventID); err != nil {
			return app.Result{}, err
		}
		// 당첨 이벤트를 아웃박스에 같은 트랜잭션으로 적재한다 → "커밋됐으면 반드시 발행".
		// dedup_id 유니크라, 어떤 이유로 두 번 실행돼도 아웃박스엔 한 건만 남는다.
		evt := domain.WinnerDetermined{EventID: eventID, UserID: userID, Seq: next, DeterminedAt: now}
		payload, err := json.Marshal(evt)
		if err != nil {
			return app.Result{}, err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO promotion_outbox (subject, event_name, payload, dedup_id)
			 VALUES ($1,$2,$3,$4) ON CONFLICT (dedup_id) DO NOTHING`,
			"promotion."+evt.EventName(), evt.EventName(), payload, evt.DedupID()); err != nil {
			return app.Result{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return app.Result{}, err
	}
	return app.Result{Seq: next, Winner: winner, Already: false}, nil
}

// EntryOf 는 사용자의 응모 상태를 조회한다(큐 모드의 상태 확인용).
func (r *PostgresRepo) EntryOf(ctx context.Context, eventID, userID string) (app.Result, bool, error) {
	var seq, target int
	err := r.pool.QueryRow(ctx,
		`SELECT e.seq, ev.target_seq
		   FROM promotion_entry e JOIN promotion_event ev ON ev.event_id = e.event_id
		  WHERE e.event_id=$1 AND e.user_id=$2`, eventID, userID).Scan(&seq, &target)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.Result{}, false, nil
	}
	if err != nil {
		return app.Result{}, false, err
	}
	return app.Result{Seq: seq, Winner: seq == target, Already: true}, true, nil
}

// DispatchOutbox 는 미발행 아웃박스 행을 잠그고(SKIP LOCKED) publish 한 뒤 발행 표시한다.
// 잠금·발행·표시가 한 트랜잭션이라, 여러 릴레이가 병렬로 돌아도 같은 행을 두 번 잡지 않는다.
func (r *PostgresRepo) DispatchOutbox(ctx context.Context, publish func(OutboxMessage) error) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT id, subject, event_name, payload, COALESCE(dedup_id, '')
		FROM promotion_outbox WHERE published_at IS NULL
		ORDER BY id LIMIT 100
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		return 0, err
	}
	var msgs []OutboxMessage
	for rows.Next() {
		var m OutboxMessage
		if err := rows.Scan(&m.ID, &m.Subject, &m.EventName, &m.Payload, &m.DedupID); err != nil {
			rows.Close()
			return 0, err
		}
		msgs = append(msgs, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var published []int64
	for _, m := range msgs {
		if err := publish(m); err != nil {
			break // 실패하면 멈추고, 커밋 안 된 나머지는 다음 주기에 다시 잠근다
		}
		published = append(published, m.ID)
	}
	if len(published) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE promotion_outbox SET published_at = now() WHERE id = ANY($1)`, published); err != nil {
			return 0, err
		}
	}
	return len(published), tx.Commit(ctx)
}

// WinnerOf 는 확정된 당첨자를 조회한다.
func (r *PostgresRepo) WinnerOf(ctx context.Context, eventID string) (string, bool, error) {
	var w *string
	err := r.pool.QueryRow(ctx,
		`SELECT winner_user_id FROM promotion_event WHERE event_id=$1`, eventID).Scan(&w)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, domain.ErrEventNotFound
	}
	if err != nil {
		return "", false, err
	}
	if w == nil {
		return "", false, nil
	}
	return *w, true, nil
}

var (
	_ app.Repository = (*PostgresRepo)(nil)
	_ OutboxStore    = (*PostgresRepo)(nil)
)
