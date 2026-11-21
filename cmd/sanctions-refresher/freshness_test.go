package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// newGaugedRefresher is a refresher whose age gauge is read back through a manual reader.
func newGaugedRefresher(t *testing.T, dir string) (*refresher, *sdkmetric.ManualReader) {
	t.Helper()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })
	g, err := newAgeGauge(mp.Meter("test"))
	require.NoError(t, err)
	r, _ := newTestRefresher(dir)
	r.age = g
	return r, reader
}

// listAges collects sanctions_list_age_seconds by list label; ok is false when the metric was never recorded.
func listAges(t *testing.T, reader *sdkmetric.ManualReader) (ages map[string]float64, unit string, ok bool) {
	t.Helper()
	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(context.Background(), &rm))
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != metricListAge {
				continue
			}
			g, isGauge := m.Data.(metricdata.Gauge[float64])
			require.True(t, isGauge, "%s must be a float64 gauge, got %T", metricListAge, m.Data)
			ages = map[string]float64{}
			for _, dp := range g.DataPoints {
				v, has := dp.Attributes.Value("list")
				require.True(t, has, "every point carries the list label")
				ages[v.AsString()] = dp.Value
			}
			return ages, m.Unit, true
		}
	}
	return nil, "", false
}

// steppingClock returns start, then start+step, start+2·step… one tick per call.
func steppingClock(start time.Time, step time.Duration) func() time.Time {
	n := 0
	return func() time.Time {
		t := start.Add(time.Duration(n) * step)
		n++
		return t
	}
}

func TestAgeGauge_Contract(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "sanctions_list_age_seconds", metricListAge)
	assert.Equal(t, 24*time.Hour, freshnessSLO)
}

func TestAgeGauge_NilIsNoop(t *testing.T) {
	t.Parallel()
	var g *ageGauge
	assert.NotPanics(t, func() { g.record(context.Background(), "ofac", time.Hour) })
}

// A refreshed list reports the age of the snapshot it just wrote: the time since its download started.
func TestFreshness_RecordsAgeAfterRefresh(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, reader := newGaugedRefresher(t, dir)
	r.now = steppingClock(testNow, 2*time.Second)

	out := r.refresh(context.Background(), job{target: mustTarget(t, "ofac"), fetch: fileFetch(fixtures["ofac"])})
	require.NoError(t, out.err)
	require.True(t, out.ageKnown)

	ages, unit, ok := listAges(t, reader)
	require.True(t, ok)
	assert.Equal(t, "s", unit)
	require.Contains(t, ages, "ofac")
	assert.InDelta(t, out.age.Seconds(), ages["ofac"], 1e-9)
	assert.Greater(t, ages["ofac"], 0.0)
	assert.Less(t, ages["ofac"], 60.0)
}

// A failed list keeps reporting the age of the snapshot still on disk, so the age grows until the SLO alert fires.
func TestFreshness_FailedRefreshReportsPreviousSnapshotAge(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		elapsed time.Duration
	}{
		{"one hour later", time.Hour},
		{"past the SLO", freshnessSLO + time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			require.NoError(t, writeFixtureSnapshot(dir, "un", testNow))

			r, reader := newGaugedRefresher(t, dir)
			r.now = func() time.Time { return testNow.Add(tt.elapsed) }
			out := r.refresh(context.Background(), job{target: mustTarget(t, "un"), fetch: failingFetch(errors.New("connection refused"))})
			require.Error(t, out.err)
			require.True(t, out.ageKnown)
			assert.Equal(t, tt.elapsed, out.age)

			ages, _, ok := listAges(t, reader)
			require.True(t, ok)
			assert.InDelta(t, tt.elapsed.Seconds(), ages["un"], 1e-9)
			assert.Equal(t, tt.elapsed > freshnessSLO, ages["un"] > freshnessSLO.Seconds())
		})
	}
}

// With no snapshot at all there is no age to report: the list is left out and the alert's absence rule covers it.
func TestFreshness_NoSnapshotRecordsNothing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	r, reader := newGaugedRefresher(t, dir)
	out := r.refresh(context.Background(), job{target: mustTarget(t, "eu"), fetch: failingFetch(errors.New("connection refused"))})
	require.Error(t, out.err)
	assert.False(t, out.ageKnown)

	ages, _, ok := listAges(t, reader)
	if ok {
		assert.NotContains(t, ages, "eu")
	}
}

// Each list is its own series.
func TestFreshness_OneSeriesPerList(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, writeFixtureSnapshot(dir, "coaf", testNow.Add(-5*time.Hour)))
	r, reader := newGaugedRefresher(t, dir)
	jobs := []job{
		{target: mustTarget(t, "ofac"), fetch: fileFetch(fixtures["ofac"])},
		{target: mustTarget(t, "coaf"), fetch: failingFetch(errors.New("connection refused"))},
	}
	require.Error(t, r.run(context.Background(), jobs))

	ages, _, ok := listAges(t, reader)
	require.True(t, ok)
	assert.Len(t, ages, 2)
	assert.InDelta(t, 0, ages["ofac"], 1e-9) // fixed clock: written and measured at the same instant
	assert.InDelta(t, (5 * time.Hour).Seconds(), ages["coaf"], 1e-9)
}

// writeFixtureSnapshot publishes the committed fixture of list as a snapshot fetched at fetchedAt.
func writeFixtureSnapshot(dir, list string, fetchedAt time.Time) error {
	r, _ := newTestRefresher(dir)
	r.now = func() time.Time { return fetchedAt }
	ts, err := parseProviders(list)
	if err != nil {
		return err
	}
	if out := r.refresh(context.Background(), job{target: ts[0], fetch: fileFetch(fixtures[list])}); out.err != nil {
		return out.err
	}
	_, err = sanctions.ReadSnapshot(dir, ts[0].source)
	return err
}
