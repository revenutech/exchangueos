package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Freshness SLO (MS-024d 1.8, RTSK.024.1208.01): every list is refreshed within 24 hours. The refresher reports,
// after each list, the age of the snapshot that list has on disk; the PrometheusRule rendered from
// deploy/helm/exchangeos/templates/sanctions-refresher-prometheusrule.yaml alerts when it exceeds 86400.
const (
	metricListAge = "sanctions_list_age_seconds"
	freshnessSLO  = 24 * time.Hour
)

// ageGauge is sanctions_list_age_seconds{list}. The metric goes out through the platform OTel pipeline
// (internal/telemetry, OTLP to the collector); the run's shutdown flushes it before the Job exits. A nil gauge
// records nothing.
type ageGauge struct{ g metric.Float64Gauge }

func newAgeGauge(m metric.Meter) (*ageGauge, error) {
	g, err := m.Float64Gauge(metricListAge,
		metric.WithUnit("s"),
		metric.WithDescription("Age of the sanctions list snapshot the screening pods load: now minus the start of the download that produced it."),
	)
	if err != nil {
		return nil, fmt.Errorf("metric %s: %w", metricListAge, err)
	}
	return &ageGauge{g: g}, nil
}

func (a *ageGauge) record(ctx context.Context, list string, age time.Duration) {
	if a == nil {
		return
	}
	a.g.Record(ctx, age.Seconds(), metric.WithAttributes(attribute.String("list", list)))
}
