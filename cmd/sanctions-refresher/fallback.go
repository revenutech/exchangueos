package main

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/revenutech/exchangeos/pkg/sanctions"
)

// Fallback rule (MS-024d 1.9, RTSK.024.1209.01). The cached snapshot is the file the previous good run left in
// --snapshot-dir; a failed refresh never touches it, so the screening pods keep matching against it. What a failure
// means for the run depends on why it failed and on how old that snapshot is:
//
//	upstream unreachable (errUpstream), snapshot ≤ 24h   fallback: Warn, the run does not fail
//	anything else, snapshot ≤ 24h                         failure: Error, the run exits non-zero (a list that
//	                                                      answers but does not parse is a format change or a
//	                                                      poisoning attempt, and needs a person; so is a redirect
//	                                                      off https, errInsecureRedirect, though the client wraps
//	                                                      it in errUpstream)
//	no snapshot, or snapshot > 24h                        blocked: Error, the run exits non-zero; the screening
//	                                                      pods refuse Match for the list (sanctions.SnapshotIndex)
//
// In every case the age keeps being reported (freshness.go), so SanctionsListStale fires once it passes 86400.

// errUpstream marks a download that did not get the list: a network error or a non-200 answer.
var errUpstream = errors.New("upstream unreachable")

// isOutage reports whether err is an upstream outage, the only failure that may fall back. A redirect off https is
// refused by the client and so carries errUpstream too, but it is an integrity signal (someone may be steering the
// download into clear text), handled like a parse failure.
func isOutage(err error) bool {
	return errors.Is(err, errUpstream) && !errors.Is(err, errInsecureRedirect)
}

// reportFailure classifies a failed refresh by the rule above, sets out.fallback or out.blocked, and logs it.
// When the list is blocked, out.err becomes a sanctions.ErrStaleSnapshot that still wraps the cause.
func (r *refresher) reportFailure(j job, out *outcome) {
	base := []zap.Field{zap.String("list", j.name), zap.Duration("duration", out.duration)}
	switch {
	case !out.ageKnown:
		out.blocked = true
		out.err = fmt.Errorf("%w: %s has no snapshot: %w", sanctions.ErrStaleSnapshot, j.name, out.err)
		r.log.Error("screening blocked: sanctions snapshot stale", append(base, zap.Error(out.err))...)
	case out.age > freshnessSLO:
		out.blocked = true
		out.err = fmt.Errorf("%w: %s snapshot is %s old (limit %s): %w",
			sanctions.ErrStaleSnapshot, j.name, out.age.Round(time.Second), freshnessSLO, out.err)
		r.log.Error("screening blocked: sanctions snapshot stale",
			append(base, zap.Duration("snapshot_age", out.age), zap.Error(out.err))...)
	case isOutage(out.err):
		out.fallback = true
		r.log.Warn("upstream unreachable, serving cached snapshot",
			append(base, zap.Duration("snapshot_age", out.age), zap.Error(out.err))...)
	default:
		r.log.Error("list refresh failed",
			append(base, zap.Duration("snapshot_age", out.age), zap.Error(out.err))...)
	}
}
