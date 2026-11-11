package sanctions

import (
	"context"
	"errors"
)

// Audit of sanctions hits (MS-024d 1.11, RTSK.024.1211.01) for the COS flow (RN_FX_039).
//
// Every entry the Cache scores for a query, above or below the query's MinScore, becomes one Hit. A Match hands all
// of them to the HitAuditor in one batch; an adapter writes each Hit as one audit_events row with
// event_type = HitEventType and schema_version = HitSchemaVersion, the Hit as the JSONB payload, and the HitScope in
// the tenant_id and correlation_id columns. That adapter is not in this package.
//
// PII: the query name is not in the event, neither in clear nor hashed. A hash of a person's name is a low-entropy
// pseudonym that a dictionary of names reverses, so it would still be personal data; the hit is tied to its
// screening request by the correlation id instead. The list entry's own name is also left out: the entry id
// identifies it, and the list data stays in the list.
//
// Retention of these events is pending DPO review and is not decided here.

// HitEventType is the audit_events.event_type of a sanctions hit.
const HitEventType = "SANCTIONS_HIT"

// HitSchemaVersion is the audit_events.schema_version of the Hit payload.
const HitSchemaVersion = "v1"

// ErrHitAudit wraps a HitAuditor failure. Match then returns no matches: a screening whose hits could not be
// audited gives no verdict.
var ErrHitAudit = errors.New("sanctions: hit audit failed")

// Hit is the payload of one SANCTIONS_HIT audit event. It holds no name.
type Hit struct {
	Source  Source `json:"source"`
	EntryID string `json:"entry_id"`
	// Score is the similarity in [0, 1] (not a monetary value).
	Score float64 `json:"score"`
	// MinScore is the threshold the query asked for.
	MinScore float64 `json:"min_score"`
	// AboveThreshold reports Score >= MinScore, i.e. whether Match returned this entry.
	AboveThreshold bool `json:"above_threshold"`
}

// HitScope is the request the hits belong to. Both fields are empty when the context carries no scope.
type HitScope struct {
	TenantID      string `json:"tenant_id"`
	CorrelationID string `json:"correlation_id"`
}

// HitAuditor records the hits of one Match. It never receives the query name. It must be safe for concurrent use.
type HitAuditor interface {
	RecordHits(ctx context.Context, scope HitScope, hits []Hit) error
}

// NopHitAuditor discards every hit. It is the Cache default.
type NopHitAuditor struct{}

// RecordHits implements HitAuditor.
func (NopHitAuditor) RecordHits(context.Context, HitScope, []Hit) error { return nil }

type hitScopeKey struct{}

// WithHitScope returns a context whose Matches audit their hits under scope.
func WithHitScope(ctx context.Context, scope HitScope) context.Context {
	return context.WithValue(ctx, hitScopeKey{}, scope)
}

// HitScopeFrom returns the scope set by WithHitScope, or the zero HitScope.
func HitScopeFrom(ctx context.Context) HitScope {
	s, _ := ctx.Value(hitScopeKey{}).(HitScope)
	return s
}

// CacheOption configures a Cache at construction.
type CacheOption func(*Cache)

// WithHitAuditor sets the auditor of the cache's hits. Nil keeps the no-op default.
func WithHitAuditor(a HitAuditor) CacheOption {
	return func(c *Cache) {
		if a != nil {
			c.auditor = a
		}
	}
}
