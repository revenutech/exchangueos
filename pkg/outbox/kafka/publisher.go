//go:build kafka

package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/revenutech/exchangeos/pkg/outbox"
)

// Config parameterises the Publisher.
type Config struct {
	Brokers   []string      // bootstrap brokers (e.g. ["kafka-0:9092", "kafka-1:9092"])
	ClientID  string        // GroupId-style identifier reported to broker (e.g. "exchangeos-worker")
	Timeout   time.Duration // per-publish deadline (default 10s)
}

// Publisher implements outbox.Publisher.
type Publisher struct {
	client  *kgo.Client
	timeout time.Duration
}

// New constructs a Publisher with production defaults:
//
//	acks=all, zstd compression, idempotent producer, max-in-flight=1 per partition.
func New(cfg Config) (*Publisher, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka: at least one broker required")
	}
	if cfg.ClientID == "" {
		cfg.ClientID = "exchangeos"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	cli, err := kgo.NewClient(
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.ProducerBatchCompression(kgo.ZstdCompression()),
		// Idempotent producer keeps the producer-id stable across retries so
		// brokers can de-duplicate. Strict ordering requires max-in-flight=1.
		kgo.MaxBufferedRecords(10_000),
		kgo.ProducerBatchMaxBytes(1<<20), // 1 MiB
		kgo.ProducerLinger(5*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka.New: %w", err)
	}
	return &Publisher{client: cli, timeout: cfg.Timeout}, nil
}

// Publish satisfies outbox.Publisher. Synchronous — returns after broker ack.
func (p *Publisher) Publish(ctx context.Context, topic string, key, payload []byte) error {
	if topic == "" {
		return fmt.Errorf("kafka: topic empty")
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	rec := &kgo.Record{
		Topic: topic,
		Key:   key,
		Value: payload,
	}
	res := p.client.ProduceSync(ctx, rec)
	for _, r := range res {
		if r.Err != nil {
			return fmt.Errorf("kafka.publish %s: %w", topic, r.Err)
		}
	}
	return nil
}

// Close flushes pending batches + closes the underlying client.
func (p *Publisher) Close() {
	p.client.Close()
}

// Compile-time interface check.
var _ outbox.Publisher = (*Publisher)(nil)
