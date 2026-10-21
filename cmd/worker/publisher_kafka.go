//go:build kafka

package main

import (
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/pkg/outbox"
	outboxkafka "github.com/revenutech/exchangeos/pkg/outbox/kafka"
)

// newPublisher (kafka tag): reads EXCHANGEOS_KAFKA_BROKERS (comma-separated) +
// EXCHANGEOS_KAFKA_CLIENT_ID and constructs a franz-go-backed Publisher.
func newPublisher(logger *zap.Logger) outbox.Publisher {
	brokers := strings.Split(os.Getenv("EXCHANGEOS_KAFKA_BROKERS"), ",")
	clientID := os.Getenv("EXCHANGEOS_KAFKA_CLIENT_ID")
	if clientID == "" {
		clientID = "exchangeos-worker"
	}
	pub, err := outboxkafka.New(outboxkafka.Config{
		Brokers:  brokers,
		ClientID: clientID,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		logger.Fatal("kafka publisher init failed", zap.Error(err))
	}
	logger.Info("kafka publisher ready", zap.Strings("brokers", brokers), zap.String("client_id", clientID))
	return pub
}

func publisherName() string { return "kafka (franz-go)" }

func closePublisher(p outbox.Publisher) {
	if k, ok := p.(*outboxkafka.Publisher); ok {
		k.Close()
	}
}
