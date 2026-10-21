# Chaos Day Runbook — Quarterly

> Full-day exercise. Block 4 hours on calendars. Quarter: YYYY-Qn.

## T-1 week — Prep

- [ ] Game-day lead schedules: 4h block + 30min retro afterwards
- [ ] Confirm staging cluster matches production topology (3 GKE nodes, 3 CRDB nodes, 3 Kafka brokers, Vault, IdentityOS)
- [ ] Confirm steady-state baseline (capture last 24h of dashboards as PNG)
- [ ] Notify `#exchangeos`, `#platform-incidents` — staging may misbehave
- [ ] Pre-load synthetic load via `k6 run tests/load/k6-trade-book.js --duration=10m` to warm caches

## T-0 — Schedule

| T+min | Activity | Scenario | Owner |
|-------|----------|----------|-------|
| 0     | Kickoff + intro + rules | — | Lead |
| 10    | CHAOS-01 pod-kill api | LOW | Observer A |
| 25    | Post-experiment writeup CHAOS-01 | — | Note-taker |
| 40    | CHAOS-02 pod-kill worker | MEDIUM | Observer A |
| 55    | Writeup CHAOS-02 | — | Note-taker |
| 70    | Break (15 min) | — | — |
| 85    | CHAOS-04 CRDB latency injection | MEDIUM | Observer B |
| 110   | Writeup CHAOS-04 | — | Note-taker |
| 125   | CHAOS-05 Kafka packet loss | HIGH | Observer B |
| 155   | Writeup CHAOS-05 + DEDUP verification | — | Note-taker + Lead |
| 175   | CHAOS-07 Vault outage (5min) | HIGH | Lead |
| 205   | Writeup CHAOS-07 | — | Note-taker |
| 220   | Wrap + assign actions | — | Lead |
| 240   | Retrospective + Slack summary | — | All |

## Per-experiment script

For each scenario:

1. **Pre-state check (T-2 min):**
   - Grafana SLO dashboard green
   - No active incidents
   - `kubectl get pods -n exchangeos` — all Ready

2. **Hypothesis statement (read aloud to room):**
   - "We expect: <X>"
   - "We will abort if: <Y>"

3. **Inject (T+0):**
   - `kubectl apply -f deploy/chaos/<file>.yaml`
   - Note start time UTC

4. **Observe:**
   - Lead watches Grafana SLO dashboard
   - Observer watches application logs (`kubectl logs -f -l app=...`)
   - Note-taker logs every event with timestamp

5. **Abort check (every 60s):**
   - Any probe failed → `kubectl delete <chaos-resource>` immediately

6. **End (T+duration):**
   - Confirm chaos resource finished
   - Capture: 5min before + during + 5min after Grafana panels

7. **Writeup (15 min):**
   - Fill `experiment-template.md` per scenario
   - Save to `docs/security/chaos/runs/YYYY-Qn-CHAOS-NN.md`

## Stop-the-day criteria

Halt the entire chaos day if any of:

- Production (not staging!) starts alerting — investigate immediately, abort all staging chaos
- Two consecutive experiments fail to meet hypothesis — something is fundamentally broken; fix first
- Observer or note-taker reports they can't keep up — slow down or postpone

## Outputs

- `docs/security/chaos/runs/YYYY-Qn/` — one MD per scenario + summary
- Action items in `.base/plans/00-governance/post-chaos-actions.md` with owners + ETA
- Slack `#exchangeos` retro thread with TL;DR + dashboard links
- Update `docs/security/iso27001-gap-tracker.md` if a resilience gap surfaced

## Annual escalation

After 4 quarterly drills in staging, schedule a chaos day in **production canary** (not full prod):
- Coordinate with customers (allow opt-out)
- Run during low-traffic window (Sunday morning UTC)
- Smaller blast radius (CHAOS-01 only)
- Have rollback ready: `task canary:rollback`

## Cite

This runbook is referenced from `docs/security/dr-runbook.md` + `docs/security/incident-response.md` + the ISO 27001 controls mapping (control 5.30 ICT readiness for business continuity).
