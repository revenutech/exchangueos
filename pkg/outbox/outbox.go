package outbox

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Record is one row of the outbox_events table.
type Record struct {
	OutboxID      uuid.UUID
	TenantID      uuid.UUID
	AggregateType string
	AggregateID   uuid.UUID
	EventName     string
	EventPayload  []byte // JSONB
	Topic         string
	PartitionKey  string
	OccurredAt    time.Time
	DispatchedAt  time.Time
	AttemptCount  int
	LastError     string
}

// Store persists outbox rows transactionally and is used by:
//   - Repository.Save implementations (Insert) — same tx as aggregate state.
//   - Worker pollers (Pending / MarkDispatched / MarkFailed).
type Store interface {
	// Insert writes a single outbox row. Implementations decide tx propagation
	// (e.g. by accepting an existing pgx.Tx via context or wrapper).
	Insert(ctx context.Context, r Record) error

	// Pending returns up to `limit` rows that have not been dispatched, ordered
	// by occurred_at ASC. Worker uses this to drive Kafka publishes.
	Pending(ctx context.Context, limit int) ([]Record, error)

	// MarkDispatched flips dispatched_at = now() for the row and optionally
	// archives it into outbox_dispatched_archive.
	MarkDispatched(ctx context.Context, outboxID uuid.UUID) error

	// MarkFailed bumps attempt_count and stores last_error (capped).
	MarkFailed(ctx context.Context, outboxID uuid.UUID, errMsg string) error
}

// Publisher abstracts the underlying Kafka client. The wire-format adapter
// (Sarama / franz-go / kgo) lives behind this interface in pkg/outbox/kafka.
//
// Implementations MUST be safe for concurrent use by the worker.
type Publisher interface {
	// Publish sends a single record to its target topic and returns when the
	// broker has acknowledged (acks=all is the recommended default).
	Publish(ctx context.Context, topic string, key, payload []byte) error
}

// PublisherFunc adapts a function to the Publisher interface (handy for tests).
type PublisherFunc func(ctx context.Context, topic string, key, payload []byte) error

func (f PublisherFunc) Publish(ctx context.Context, topic string, key, payload []byte) error {
	return f(ctx, topic, key, payload)
}

// Sentinel errors.
var (
	ErrEmpty        = errors.New("outbox: no pending records")
	ErrTopicMissing = errors.New("outbox: topic missing on record")
	ErrPublishFail  = errors.New("outbox: publisher failed")
)

// Dispatch is the worker loop helper: take one Pending batch, publish each,
// mark dispatched or failed. Returns count dispatched + first error encountered.
//
// Workers SHOULD wrap Dispatch in a loop with backoff between batches.
func Dispatch(ctx context.Context, store Store, pub Publisher, batchSize int) (int, error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	pending, err := store.Pending(ctx, batchSize)
	if err != nil {
		return 0, err
	}
	dispatched := 0
	var firstErr error
	for _, r := range pending {
		if r.Topic == "" {
			_ = store.MarkFailed(ctx, r.OutboxID, ErrTopicMissing.Error())
			if firstErr == nil {
				firstErr = ErrTopicMissing
			}
			continue
		}
		key := []byte(r.PartitionKey)
		if len(key) == 0 {
			key = []byte(r.AggregateID.String())
		}
		if err := pub.Publish(ctx, r.Topic, key, r.EventPayload); err != nil {
			_ = store.MarkFailed(ctx, r.OutboxID, err.Error())
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if err := store.MarkDispatched(ctx, r.OutboxID); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		dispatched++
	}
	return dispatched, firstErr
}
