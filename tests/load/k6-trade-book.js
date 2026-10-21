// tests/load/k6-trade-book.js — sustained-load + soak test for exchangeos-api.
//
// Run via `k6 run tests/load/k6-trade-book.js`.
// Use with `k6 cloud` against a non-production tenant for true production capacity tests.
//
// SLO targets (mirror Argo Rollouts AnalysisTemplate gates):
//   - 5xx rate < 1%
//   - p99 latency < 500ms
//   - error rate (k6 failed checks) < 1%

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// ── Custom metrics ──────────────────────────────────────────────────────────
const errors = new Rate('errors');
const latencyHealthz = new Trend('latency_healthz');
const latencyCurrencies = new Trend('latency_currencies');

// ── Scenarios + thresholds (SLO-aligned) ────────────────────────────────────
export const options = {
  scenarios: {
    // Warmup
    warmup: {
      executor: 'constant-vus',
      vus: 5,
      duration: '30s',
      gracefulStop: '5s',
    },
    // Sustained load
    sustained: {
      executor: 'ramping-vus',
      startTime: '35s',
      startVUs: 10,
      stages: [
        { duration: '1m',  target: 50 },
        { duration: '3m',  target: 100 },
        { duration: '5m',  target: 100 },
        { duration: '1m',  target: 10 },
      ],
      gracefulRampDown: '10s',
    },
  },
  thresholds: {
    'http_req_failed':     ['rate<0.01'],   // < 1% 5xx
    'http_req_duration{endpoint:healthz}':    ['p(99)<200'],
    'http_req_duration{endpoint:currencies}': ['p(99)<500'],
    'errors':              ['rate<0.01'],
  },
};

const BASE_URL = __ENV.EXCHANGEOS_BASE_URL || 'http://localhost:8084';

export default function () {
  // 1. health probe
  let res = http.get(`${BASE_URL}/healthz`, {
    tags: { endpoint: 'healthz' },
  });
  check(res, { 'healthz 200': r => r.status === 200 }) || errors.add(1);
  latencyHealthz.add(res.timings.duration);

  // 2. refdata read
  res = http.get(`${BASE_URL}/v1/refdata/currencies?active_only=true`, {
    tags: { endpoint: 'currencies' },
  });
  check(res, {
    'currencies 200': r => r.status === 200,
    'currencies has body': r => r.body.length > 0,
  }) || errors.add(1);
  latencyCurrencies.add(res.timings.duration);

  // 3. version (cheap; used as keepalive)
  res = http.get(`${BASE_URL}/version`, { tags: { endpoint: 'version' } });
  check(res, { 'version 200': r => r.status === 200 }) || errors.add(1);

  // 4. negative test — non-existent trade returns 404 (verifies error path mapping under load)
  res = http.get(`${BASE_URL}/v1/trades/00000000-0000-0000-0000-000000000001`, {
    tags: { endpoint: 'trade_404' },
  });
  check(res, { 'unknown trade 404': r => r.status === 404 }) || errors.add(1);

  sleep(1);
}

// ── Summary handler — pretty + machine-readable output ──────────────────────
export function handleSummary(data) {
  return {
    'stdout': JSON.stringify(
      {
        vus_max:       data.metrics.vus_max?.values.max,
        http_reqs:     data.metrics.http_reqs?.values.count,
        http_req_failed_rate: data.metrics.http_req_failed?.values.rate,
        p99_healthz:   data.metrics['http_req_duration{endpoint:healthz}']?.values?.['p(99)'],
        p99_currencies: data.metrics['http_req_duration{endpoint:currencies}']?.values?.['p(99)'],
      },
      null, 2,
    ) + '\n',
    'tests/load/results-last.json': JSON.stringify(data, null, 2),
  };
}
