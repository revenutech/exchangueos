//go:build !kafka

package main

import (
	"context"

	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/pkg/outbox"
)

// newPublisher (default tag): logs each publish and returns nil. Use -tags kafka
// for a real Kafka publisher.
func newPublisher(logger *zap.Logger) outbox.Publisher {
	return outbox.PublisherFunc(func(_ context.Context, topic string, key, payload []byte) error {
		logger.Info("outbox.publish (noop)",
			zap.String("topic", topic),
			zap.Int("key_len", len(key)),
			zap.Int("payload_len", len(payload)),
		)
		return nil
	})
}

func publisherName() string { return "noop (logs only)" }
func closePublisher(_ outbox.Publisher) {}
