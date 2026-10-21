package outbox_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/pkg/outbox"
)

// fakeStore — minimal in-memory Store for testing Dispatch.
type fakeStore struct {
	mu          sync.Mutex
	pending     []outbox.Record
	dispatched  []uuid.UUID
	failed      map[uuid.UUID]string
}

func newFakeStore(rs ...outbox.Record) *fakeStore {
	return &fakeStore{pending: rs, failed: map[uuid.UUID]string{}}
}

func (s *fakeStore) Insert(_ context.Context, r outbox.Record) error {
	s.mu.Lock()
	s.pending = append(s.pending, r)
	s.mu.Unlock()
	return nil
}

func (s *fakeStore) Pending(_ context.Context, limit int) ([]outbox.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if limit > len(s.pending) {
		limit = len(s.pending)
	}
	out := append([]outbox.Record(nil), s.pending[:limit]...)
	return out, nil
}

func (s *fakeStore) MarkDispatched(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatched = append(s.dispatched, id)
	// remove from pending
	s.pending = removeBy(s.pending, id)
	return nil
}

func (s *fakeStore) MarkFailed(_ context.Context, id uuid.UUID, errMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failed[id] = errMsg
	return nil
}

func removeBy(rs []outbox.Record, id uuid.UUID) []outbox.Record {
	out := rs[:0]
	for _, r := range rs {
		if r.OutboxID != id {
			out = append(out, r)
		}
	}
	return out
}

func sampleRecord(topic string) outbox.Record {
	return outbox.Record{
		OutboxID:      uuid.New(),
		TenantID:      uuid.New(),
		AggregateType: "Trade",
		AggregateID:   uuid.New(),
		EventName:     "trade.created.v1",
		EventPayload:  []byte(`{"x":1}`),
		Topic:         topic,
		OccurredAt:    time.Now().UTC(),
	}
}

func TestDispatch_HappyPath(t *testing.T) {
	r1 := sampleRecord("exchangeos.trade")
	r2 := sampleRecord("exchangeos.trade")
	store := newFakeStore(r1, r2)

	var got []string
	pub := outbox.PublisherFunc(func(_ context.Context, topic string, key, payload []byte) error {
		got = append(got, topic+":"+string(payload))
		return nil
	})

	n, err := outbox.Dispatch(context.Background(), store, pub, 100)
	if err != nil {
		t.Fatalf("Dispatch: %v", err)
	}
	if n != 2 {
		t.Fatalf("dispatched: got %d want 2", n)
	}
	if len(got) != 2 {
		t.Fatalf("published count: %d", len(got))
	}
	if len(store.dispatched) != 2 || len(store.pending) != 0 {
		t.Fatalf("store state: dispatched=%d pending=%d", len(store.dispatched), len(store.pending))
	}
}

func TestDispatch_PublishError_MarksFailed(t *testing.T) {
	r := sampleRecord("exchangeos.trade")
	store := newFakeStore(r)
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _ []byte, _ []byte) error {
		return errors.New("broker down")
	})
	n, err := outbox.Dispatch(context.Background(), store, pub, 10)
	if err == nil {
		t.Fatal("expected publisher error returned")
	}
	if n != 0 {
		t.Fatalf("dispatched: got %d want 0", n)
	}
	if store.failed[r.OutboxID] != "broker down" {
		t.Fatalf("failed map missing: %v", store.failed)
	}
}

func TestDispatch_MissingTopic_MarksFailed(t *testing.T) {
	r := sampleRecord("") // no topic
	store := newFakeStore(r)
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _ []byte, _ []byte) error {
		t.Fatal("publisher should not be called for missing topic")
		return nil
	})
	n, err := outbox.Dispatch(context.Background(), store, pub, 10)
	if !errors.Is(err, outbox.ErrTopicMissing) {
		t.Fatalf("want ErrTopicMissing, got %v", err)
	}
	if n != 0 {
		t.Fatalf("dispatched: %d", n)
	}
}

func TestDispatch_EmptyBatch_NoError(t *testing.T) {
	store := newFakeStore()
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _ []byte, _ []byte) error { return nil })
	n, err := outbox.Dispatch(context.Background(), store, pub, 10)
	if err != nil {
		t.Fatalf("Dispatch on empty: %v", err)
	}
	if n != 0 {
		t.Fatalf("dispatched on empty: %d", n)
	}
}

func TestDispatch_DefaultBatchSize(t *testing.T) {
	store := newFakeStore()
	pub := outbox.PublisherFunc(func(_ context.Context, _ string, _ []byte, _ []byte) error { return nil })
	if _, err := outbox.Dispatch(context.Background(), store, pub, 0); err != nil {
		t.Fatalf("Dispatch with batchSize=0: %v", err)
	}
}
