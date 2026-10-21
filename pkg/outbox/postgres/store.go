// Package postgres — pgx/v5 backed outbox.Store against migration 000009.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/revenutech/exchangeos/pkg/outbox"
)

type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Insert writes one outbox row. Designed to be called from inside an existing
// repository transaction — for now uses the pool directly; future refactor
// accepts a pgx.Tx via context propagation.
func (s *Store) Insert(ctx context.Context, r outbox.Record) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO outbox_events (
			outbox_id, tenant_id, aggregate_type, aggregate_id,
			event_name, event_payload, topic, partition_key, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		nilToNew(r.OutboxID), r.TenantID, r.AggregateType, r.AggregateID,
		r.EventName, r.EventPayload, r.Topic,
		nullableString(r.PartitionKey), r.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("outbox.insert: %w", err)
	}
	return nil
}

// Pending returns up to `limit` undispatched rows ordered by occurred_at ASC.
func (s *Store) Pending(ctx context.Context, limit int) ([]outbox.Record, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT outbox_id, tenant_id, aggregate_type, aggregate_id,
		       event_name, event_payload, topic, COALESCE(partition_key, ''),
		       occurred_at, attempt_count, COALESCE(last_error, '')
		FROM outbox_events
		WHERE dispatched_at IS NULL
		ORDER BY occurred_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("outbox.pending: %w", err)
	}
	defer rows.Close()

	out := make([]outbox.Record, 0, limit)
	for rows.Next() {
		var r outbox.Record
		if err := rows.Scan(
			&r.OutboxID, &r.TenantID, &r.AggregateType, &r.AggregateID,
			&r.EventName, &r.EventPayload, &r.Topic, &r.PartitionKey,
			&r.OccurredAt, &r.AttemptCount, &r.LastError,
		); err != nil {
			return nil, fmt.Errorf("outbox.scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// MarkDispatched flips dispatched_at = now() and copies the row to the archive
// table (best-effort — archive failure does not roll back).
func (s *Store) MarkDispatched(ctx context.Context, id uuid.UUID) error {
	cmd, err := s.pool.Exec(ctx, `
		UPDATE outbox_events SET dispatched_at = current_timestamp()
		WHERE outbox_id = $1 AND dispatched_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("outbox.mark_dispatched: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("outbox.mark_dispatched: no row affected for %s", id)
	}
	// Archive (best-effort).
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO outbox_dispatched_archive (
			outbox_id, tenant_id, aggregate_type, aggregate_id,
			event_name, topic, partition_key, occurred_at, dispatched_at, attempt_count
		)
		SELECT outbox_id, tenant_id, aggregate_type, aggregate_id,
		       event_name, topic, COALESCE(partition_key, ''),
		       occurred_at, dispatched_at, attempt_count
		FROM outbox_events WHERE outbox_id = $1
		ON CONFLICT (outbox_id) DO NOTHING`, id); err != nil {
		// Log via caller — do not roll back the dispatch flag.
		_ = err
	}
	return nil
}

// MarkFailed bumps attempt_count + stores last_error (truncated at 512 chars).
func (s *Store) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	if len(errMsg) > 512 {
		errMsg = errMsg[:512]
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE outbox_events
		SET attempt_count = attempt_count + 1, last_error = $2
		WHERE outbox_id = $1`, id, errMsg)
	if err != nil {
		return fmt.Errorf("outbox.mark_failed: %w", err)
	}
	return nil
}

// nilToNew returns the input UUID unless it is uuid.Nil — in which case a fresh
// UUIDv4 is generated. Lets callers Insert without pre-generating ids.
func nilToNew(id uuid.UUID) uuid.UUID {
	if id == uuid.Nil {
		return uuid.New()
	}
	return id
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Helper: silence the pgx import lint when only error type is needed.
var _ = errors.New
var _ = pgx.ErrNoRows
