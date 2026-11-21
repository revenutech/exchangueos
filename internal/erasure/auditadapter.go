// auditadapter.go — pgxpool-backed AuditEmitter writing per-op rows into
// audit_events and a single completion event into outbox_events.
//
// The audit rows are evidence per ISO 27001 control 8.15 (logging) and feed
// the LGPD audit bundle. The outbox row drives the downstream Kafka topic
// `exchangeos.lgpd.erasure.events.v1` so LedgerOS / ComplOS can apply
// equivalent erasure to their own datasets.
package erasure

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erasure event constants — keep the wire format stable.
const (
	auditSource         = "erasure-worker"
	auditSchemaVersion  = "v1"
	auditOpEventType    = "LGPD_ERASURE_OP"
	auditDoneEventType  = "LGPD_ERASURE_COMPLETED"
	outboxAggregateType = "LGPDErasure"
	outboxTopic         = "exchangeos.lgpd.erasure.events.v1"
	outboxEventName     = "lgpd.erasure_completed.v1"
)

// PgxAudit implements AuditEmitter against a pgxpool.Pool. Operator tenant
// owns the audit trail and the outbox event — see docs/security/data-
// lifecycle/erasure-workflow.md Stage 3 for the rationale.
type PgxAudit struct {
	pool           *pgxpool.Pool
	operatorTenant uuid.UUID
}

// NewPgxAudit constructs the adapter. operatorTenant is the tenant_id of the
// regulatory operator (typically the platform tenant) — required because the
// audit_events + outbox_events tables both carry a NOT NULL tenant_id FK.
func NewPgxAudit(pool *pgxpool.Pool, operatorTenant uuid.UUID) *PgxAudit {
	return &PgxAudit{pool: pool, operatorTenant: operatorTenant}
}

// EmitOp writes one audit_events row per Operation. The full OpAudit struct
// is serialised into the JSONB payload so downstream auditors can replay
// state without needing to inspect related tables.
func (a *PgxAudit) EmitOp(ctx context.Context, evt OpAudit) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("audit.op.marshal: %w", err)
	}
	_, err = a.pool.Exec(ctx, `
		INSERT INTO audit_events (
			tenant_id, correlation_id, source, event_type, schema_version,
			payload, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.operatorTenant, evt.Ticket, auditSource, auditOpEventType, auditSchemaVersion,
		payload, evt.AppliedAt,
	)
	if err != nil {
		return fmt.Errorf("audit.op.insert: %w", err)
	}
	return nil
}

// EmitCompletion writes both a final audit row (LGPD_ERASURE_COMPLETED) and an
// outbox row (lgpd.erasure_completed.v1) atomically inside one tx — either
// both land or neither does. The Kafka dispatcher picks up the outbox row
// asynchronously; downstream consumers correlate by the ticket id.
func (a *PgxAudit) EmitCompletion(ctx context.Context, evt CompletionAudit) error {
	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("audit.completion.marshal: %w", err)
	}
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("audit.completion.tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (
			tenant_id, correlation_id, source, event_type, schema_version,
			payload, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		a.operatorTenant, evt.Ticket, auditSource, auditDoneEventType, auditSchemaVersion,
		payload, evt.CompletedAt,
	); err != nil {
		return fmt.Errorf("audit.completion.insert: %w", err)
	}

	aggregateID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("lgpd-erasure:"+evt.Ticket))
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (
			tenant_id, aggregate_type, aggregate_id, event_name,
			event_payload, topic, partition_key, occurred_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		a.operatorTenant, outboxAggregateType, aggregateID, outboxEventName,
		payload, outboxTopic, evt.Ticket, evt.CompletedAt,
	); err != nil {
		return fmt.Errorf("outbox.completion.insert: %w", err)
	}
	return tx.Commit(ctx)
}
