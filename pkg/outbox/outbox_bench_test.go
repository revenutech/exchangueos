package outbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/pkg/outbox"
)

// BenchmarkDispatch_HotPath measures the per-record overhead of the Dispatch
// loop assuming a no-op Publisher (isolates the loop + store interactions
// from broker round-trip time). Target: < 50µs per record on commodity hardware.
func BenchmarkDispatch_HotPath(b *testing.B) {
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _, _ []byte) error {
		return nil
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := newFakeStore(sampleRecord("exchangeos.trade.events"))
		_, _ = outbox.Dispatch(context.Background(), store, pub, 1)
	}
}

// BenchmarkDispatch_Batch100 measures throughput on a 100-record batch.
// Target: < 5ms per batch (50µs × 100 records).
func BenchmarkDispatch_Batch100(b *testing.B) {
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _, _ []byte) error {
		return nil
	})
	records := make([]outbox.Record, 100)
	for i := range records {
		records[i] = sampleRecord("exchangeos.trade.events")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := newFakeStore(records...)
		_, _ = outbox.Dispatch(context.Background(), store, pub, 100)
	}
}

// BenchmarkRecord_Build measures the construction cost of an outbox.Record value
// (mostly UUID allocations + struct field init).
func BenchmarkRecord_Build(b *testing.B) {
	tenantID := uuid.New()
	now := time.Now().UTC()
	payload := []byte(`{"trade_id":"abc","amount":"1000000.00"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = outbox.Record{
			OutboxID:      uuid.New(),
			TenantID:      tenantID,
			AggregateType: "Trade",
			AggregateID:   uuid.New(),
			EventName:     "trade.created.v1",
			EventPayload:  payload,
			Topic:         "exchangeos.trade.events",
			OccurredAt:    now,
		}
	}
}
