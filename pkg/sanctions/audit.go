package sanctions

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Audit of sanctions hits (MS-024d 1.11, RTSK.024.1211.01) for the COS flow (RN_FX_039).
//
// Every entry the Cache scores for a query is audited. An entry at or above the query's MinScore becomes one Hit,
// in detail (source, entry id, score): the COS flow needs it. The entries below MinScore are only aggregated, as
// their count and best score, with no source nor entry id per entry: anyone holding the public list and a
// dictionary of names could replay candidates through the same matcher and compare the (entry id, score) vector,
// so below the threshold that vector is a quasi-identifier of the query. Recording sub-threshold detail needs an
// explicit DPO decision, together with retention.
//
// A Match that scored anything hands one HitBatch to the HitAuditor; an adapter writes it as one audit_events row
// with event_type = HitEventType and schema_version = HitSchemaVersion, the batch (without its scope) as the JSONB
// payload, and the HitScope in the tenant_id and correlation_id columns. That adapter is not in this package.
//
// Tenant, fail-closed: with an auditor wired, a Match whose context carries no HitScope.TenantID fails with
// ErrHitAudit before scoring, so no row is ever written without a tenant. Match without a scoped context
// (Cache.Match, or MatchContext on a context without WithHitScope) is therefore not a production path; it only works
// with the no-op default, where there is nothing to audit.
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

// ErrHitAudit wraps a HitAuditor failure, or reports a missing tenant in the HitScope. Match then returns no
// matches: a screening whose hits could not be audited gives no verdict.
var ErrHitAudit = errors.New("sanctions: hit audit failed")

// Hit is one entry at or above the threshold, i.e. one entry Match returned. It holds no name.
type Hit struct {
	Source  Source `json:"source"`
	EntryID string `json:"entry_id"`
	// Score is the similarity in [0, 1] (not a monetary value).
	Score float64 `json:"score"`
}

// SubThreshold aggregates the entries scored below the threshold. It deliberately holds no source nor entry id.
type SubThreshold struct {
	Count int `json:"count"`
	// BestScore is the highest score below the threshold; zero when Count is zero.
	BestScore float64 `json:"best_score"`
}

// HitBatch is the audit of one Match: one SANCTIONS_HIT event. It holds no name.
type HitBatch struct {
	Scope HitScope `json:"scope"`
	// MinScore is the threshold the query asked for.
	MinScore       float64      `json:"min_score"`
	Hits           []Hit        `json:"hits"`
	BelowThreshold SubThreshold `json:"below_threshold"`
}

// HitScope is the request the hits belong to. Both fields are empty when the context carries no scope.
type HitScope struct {
	TenantID      string `json:"tenant_id"`
	CorrelationID string `json:"correlation_id"`
}

// HitAuditor records the audit of one Match. It never receives the query name, and the Cache only calls it with a
// non-blank Scope.TenantID. It must be safe for concurrent use.
type HitAuditor interface {
	RecordHits(ctx context.Context, batch HitBatch) error
}

// NopHitAuditor discards every batch. A Cache without an auditor behaves like it.
type NopHitAuditor struct{}

// RecordHits implements HitAuditor.
func (NopHitAuditor) RecordHits(context.Context, HitBatch) error { return nil }

type hitScopeKey struct{}

// WithHitScope returns a context whose Matches audit their hits under scope.
func WithHitScope(ctx context.Context, scope HitScope) context.Context {
	return context.WithValue(ctx, hitScopeKey{}, scope)
}

// auditScope returns the context's scope, or ErrHitAudit when it has no tenant.
func auditScope(ctx context.Context) (HitScope, error) {
	s := HitScopeFrom(ctx)
	if strings.TrimSpace(s.TenantID) == "" {
		return HitScope{}, fmt.Errorf("%w: no tenant in the hit scope", ErrHitAudit)
	}
	return s, nil
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
