package domain_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/modules/compliance/domain"
)

// plantedNarrative is PII that must never leave the aggregate through fmt, slog or JSON.
const plantedNarrative = "Fulano de Tal CPF 123.456.789-09 depositou em especie"

var cosT0 = time.Date(2026, 11, 10, 14, 0, 0, 0, time.UTC)

func newCOSInput() domain.NewCOSCaseInput {
	return domain.NewCOSCaseInput{
		TenantID:          uuid.New(),
		CounterpartyID:    uuid.New(),
		ScreeningResultID: uuid.New(),
		CreatedBy:         "analyst-1",
		Narrative:         plantedNarrative,
		At:                cosT0,
	}
}

func newCOS(t *testing.T) *domain.COSCase {
	t.Helper()
	c, err := domain.NewCOSCase(newCOSInput())
	require.NoError(t, err)
	return c
}

// approved drives a case to APPROVED with two distinct approvers, neither the creator.
func approved(t *testing.T) *domain.COSCase {
	t.Helper()
	c := newCOS(t)
	require.NoError(t, c.SubmitForReview(cosT0.Add(time.Minute)))
	require.NoError(t, c.Approve("officer-a", cosT0.Add(2*time.Minute)))
	require.NoError(t, c.Approve("officer-b", cosT0.Add(3*time.Minute)))
	return c
}

func TestNewCOSCase_Happy(t *testing.T) {
	t.Parallel()
	in := newCOSInput()
	c, err := domain.NewCOSCase(in)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID())
	assert.Equal(t, in.TenantID, c.TenantID())
	assert.Equal(t, in.CounterpartyID, c.CounterpartyID())
	assert.Equal(t, in.ScreeningResultID, c.ScreeningResultID())
	assert.Equal(t, "analyst-1", c.CreatedBy())
	assert.Equal(t, plantedNarrative, c.Narrative())
	assert.Equal(t, domain.COSDraft, c.Status())
	assert.Equal(t, 1, c.Version())
	assert.Empty(t, c.Approvers())

	ev := c.PendingEvents()
	require.Len(t, ev, 1)
	assert.Equal(t, "compliance.cos_case.drafted.v1", ev[0].EventName())
	c.MarkEventsCommitted()
	assert.Empty(t, c.PendingEvents())
}

func TestNewCOSCase_ScreeningResultOptional(t *testing.T) {
	t.Parallel()
	in := newCOSInput()
	in.ScreeningResultID = uuid.Nil
	c, err := domain.NewCOSCase(in)
	require.NoError(t, err)
	assert.Equal(t, uuid.Nil, c.ScreeningResultID())
}

func TestNewCOSCase_BadInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*domain.NewCOSCaseInput)
	}{
		{"nil tenant", func(in *domain.NewCOSCaseInput) { in.TenantID = uuid.Nil }},
		{"nil counterparty", func(in *domain.NewCOSCaseInput) { in.CounterpartyID = uuid.Nil }},
		{"empty creator", func(in *domain.NewCOSCaseInput) { in.CreatedBy = "  " }},
		{"empty narrative", func(in *domain.NewCOSCaseInput) { in.Narrative = " " }},
		{"zero time", func(in *domain.NewCOSCaseInput) { in.At = time.Time{} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			in := newCOSInput()
			tc.mutate(&in)
			_, err := domain.NewCOSCase(in)
			require.ErrorIs(t, err, domain.ErrInvalidInput)
		})
	}
}

func TestRN_FX_039_COSCase_Lifecycle_Accepted(t *testing.T) {
	t.Parallel()
	c := approved(t)
	assert.Equal(t, domain.COSApproved, c.Status())
	assert.Equal(t, cosT0.Add(3*time.Minute), c.DecidedAt(), "RN_FX_039 counts from the decision")
	assert.ElementsMatch(t, []string{"officer-a", "officer-b"}, c.Approvers())

	require.NoError(t, c.MarkSubmitted(cosT0.Add(time.Hour), "SISCOAF-PROTO-1"))
	assert.Equal(t, domain.COSSubmitted, c.Status())
	assert.Equal(t, "SISCOAF-PROTO-1", c.ProtocolID())
	assert.Equal(t, cosT0.Add(time.Hour), c.SubmittedAt())

	require.NoError(t, c.MarkAccepted(cosT0.Add(2*time.Hour)))
	assert.Equal(t, domain.COSAccepted, c.Status())
	assert.Equal(t, 6, c.Version(), "new + review + 2 approvals + submit + accept")

	var names []string
	for _, e := range c.PendingEvents() {
		names = append(names, e.EventName())
		assert.Equal(t, c.TenantID(), e.TenantID(), "tenant on %s", e.EventName())
	}
	assert.Equal(t, []string{
		"compliance.cos_case.drafted.v1",
		"compliance.cos_case.review_requested.v1",
		"compliance.cos_case.approval_recorded.v1",
		"compliance.cos_case.approval_recorded.v1",
		"compliance.cos_case.approved.v1",
		"compliance.cos_case.submitted.v1",
		"compliance.cos_case.accepted.v1",
	}, names)
}

func TestCOSCase_Lifecycle_Rejected(t *testing.T) {
	t.Parallel()
	c := approved(t)
	require.NoError(t, c.MarkSubmitted(cosT0.Add(time.Hour), "SISCOAF-PROTO-2"))
	require.ErrorIs(t, c.MarkRejected(cosT0.Add(2*time.Hour), " "), domain.ErrInvalidInput)
	require.NoError(t, c.MarkRejected(cosT0.Add(2*time.Hour), "layout invalido"))
	assert.Equal(t, domain.COSRejected, c.Status())
	assert.Equal(t, "layout invalido", c.RejectionReason())
	assert.Equal(t, cosT0.Add(2*time.Hour), c.RespondedAt())
	for _, e := range c.PendingEvents() {
		assert.Equal(t, c.TenantID(), e.TenantID(), "tenant on %s", e.EventName())
	}
}

func TestCOSCase_InvalidTransitions(t *testing.T) {
	t.Parallel()
	at := cosT0.Add(time.Hour)
	steps := map[string]func(*domain.COSCase) error{
		"review":  func(c *domain.COSCase) error { return c.SubmitForReview(at) },
		"approve": func(c *domain.COSCase) error { return c.Approve("officer-z", at) },
		"submit":  func(c *domain.COSCase) error { return c.MarkSubmitted(at, "P") },
		"accept":  func(c *domain.COSCase) error { return c.MarkAccepted(at) },
		"reject":  func(c *domain.COSCase) error { return c.MarkRejected(at, "r") },
	}
	states := map[string]func(*testing.T) *domain.COSCase{
		"DRAFT": newCOS,
		"UNDER_REVIEW": func(t *testing.T) *domain.COSCase {
			c := newCOS(t)
			require.NoError(t, c.SubmitForReview(at))
			return c
		},
		"APPROVED": approved,
		"SUBMITTED": func(t *testing.T) *domain.COSCase {
			c := approved(t)
			require.NoError(t, c.MarkSubmitted(at, "P"))
			return c
		},
		"ACCEPTED": func(t *testing.T) *domain.COSCase {
			c := approved(t)
			require.NoError(t, c.MarkSubmitted(at, "P"))
			require.NoError(t, c.MarkAccepted(at))
			return c
		},
		"REJECTED": func(t *testing.T) *domain.COSCase {
			c := approved(t)
			require.NoError(t, c.MarkSubmitted(at, "P"))
			require.NoError(t, c.MarkRejected(at, "r"))
			return c
		},
	}
	allowed := map[string]string{
		"DRAFT":        "review",
		"UNDER_REVIEW": "approve",
		"APPROVED":     "submit",
		"SUBMITTED":    "accept,reject",
	}
	for state, mk := range states {
		for step, do := range steps {
			if slices.Contains(strings.Split(allowed[state], ","), step) {
				continue
			}
			t.Run(state+"/"+step, func(t *testing.T) {
				t.Parallel()
				c := mk(t)
				v := c.Version()
				err := do(c)
				require.ErrorIs(t, err, domain.ErrInvalidTransition)
				assert.Equal(t, state, string(c.Status()), "state unchanged")
				assert.Equal(t, v, c.Version(), "version unchanged")
			})
		}
	}
}

func TestCOSCase_FourEyes(t *testing.T) {
	t.Parallel()
	at := cosT0.Add(time.Hour)
	inReview := func(t *testing.T) *domain.COSCase {
		c := newCOS(t)
		require.NoError(t, c.SubmitForReview(at))
		return c
	}

	t.Run("one approver does not approve nor allow submit", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.NoError(t, c.Approve("officer-a", at))
		assert.Equal(t, domain.COSUnderReview, c.Status())
		assert.True(t, c.DecidedAt().IsZero())
		require.ErrorIs(t, c.MarkSubmitted(at, "P"), domain.ErrInvalidTransition)
	})
	t.Run("same approver twice does not count", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.NoError(t, c.Approve("officer-a", at))
		require.ErrorIs(t, c.Approve("officer-a", at), domain.ErrDuplicateApprover)
		require.ErrorIs(t, c.Approve(" officer-a ", at), domain.ErrDuplicateApprover, "trimmed identity")
		assert.Equal(t, domain.COSUnderReview, c.Status())
		assert.Equal(t, []string{"officer-a"}, c.Approvers())
	})
	t.Run("creator cannot approve", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.ErrorIs(t, c.Approve("analyst-1", at), domain.ErrSelfApproval)
		require.NoError(t, c.Approve("officer-a", at))
		require.ErrorIs(t, c.Approve("analyst-1", at), domain.ErrSelfApproval)
		assert.Equal(t, domain.COSUnderReview, c.Status())
	})
	t.Run("empty approver refused", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.ErrorIs(t, c.Approve(" ", at), domain.ErrInvalidInput)
		assert.Empty(t, c.Approvers())
	})
	t.Run("two distinct non-creator approvers approve", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.NoError(t, c.Approve("officer-a", at))
		require.NoError(t, c.Approve("officer-b", at.Add(time.Minute)))
		assert.Equal(t, domain.COSApproved, c.Status())
		assert.Equal(t, at.Add(time.Minute), c.DecidedAt())
		assert.Equal(t, domain.COSRequiredApprovers, len(c.Approvers()))
	})
	t.Run("approvers slice is a copy", func(t *testing.T) {
		t.Parallel()
		c := inReview(t)
		require.NoError(t, c.Approve("officer-a", at))
		got := c.Approvers()
		got[0] = "tampered"
		assert.Equal(t, []string{"officer-a"}, c.Approvers())
	})
}

func TestCOSCase_Redaction(t *testing.T) {
	t.Parallel()
	c := approved(t)
	require.NoError(t, c.MarkSubmitted(cosT0.Add(time.Hour), "P"))
	require.NoError(t, c.MarkRejected(cosT0.Add(2*time.Hour), plantedNarrative))

	// Control: the planted string is detectable when a struct does print it.
	require.Contains(t, fmt.Sprintf("%+v", struct{ N string }{plantedNarrative}), plantedNarrative)

	outputs := map[string]string{}
	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		outputs["ptr "+verb] = fmt.Sprintf(verb, c)
		outputs["val "+verb] = fmt.Sprintf(verb, *c)
	}
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("cos", slog.Any("case", c), slog.Any("case_val", *c))
	outputs["slog json"] = buf.String()
	var tbuf bytes.Buffer
	slog.New(slog.NewTextHandler(&tbuf, nil)).Info("cos", slog.Any("case", c))
	outputs["slog text"] = tbuf.String()
	j, err := json.Marshal(c)
	require.NoError(t, err)
	outputs["json ptr"] = string(j)
	jv, err := json.Marshal(*c)
	require.NoError(t, err)
	outputs["json val"] = string(jv)
	for _, e := range c.PendingEvents() {
		outputs["event "+e.EventName()] += fmt.Sprintf("%+v %#v", e, e)
	}

	for name, out := range outputs {
		assert.NotContains(t, out, "Fulano", name)
		assert.NotContains(t, out, "123.456.789-09", name)
		assert.NotContains(t, out, "especie", name)
	}
	assert.Contains(t, outputs["ptr %v"], c.ID().String())
	assert.Contains(t, outputs["ptr %v"], "REJECTED")
	assert.Contains(t, outputs["slog json"], fmt.Sprintf(`"narrative_len":%d`, len(plantedNarrative)))
	assert.Contains(t, outputs["json ptr"], `"status":"REJECTED"`)
	assert.Contains(t, outputs["json ptr"], fmt.Sprintf(`"rejection_len":%d`, len(plantedNarrative)))
}

func TestCOSCase_ErrorsAreDistinct(t *testing.T) {
	t.Parallel()
	assert.False(t, errors.Is(domain.ErrSelfApproval, domain.ErrDuplicateApprover))
	assert.ErrorIs(t, domain.ErrSelfApproval, domain.ErrInvalidInput)
	assert.ErrorIs(t, domain.ErrDuplicateApprover, domain.ErrInvalidInput)
}

// RN_FX_039 counts from DecidedAt: a zero instant would break the deadline, so every transition refuses it.
func TestCOSCase_ZeroTimeRefused(t *testing.T) {
	t.Parallel()
	var zero time.Time
	at := cosT0.Add(time.Hour)
	cases := []struct {
		name string
		mk   func(*testing.T) *domain.COSCase
		do   func(*domain.COSCase) error
	}{
		{"review", newCOS, func(c *domain.COSCase) error { return c.SubmitForReview(zero) }},
		{"approve", func(t *testing.T) *domain.COSCase {
			c := newCOS(t)
			require.NoError(t, c.SubmitForReview(at))
			return c
		}, func(c *domain.COSCase) error { return c.Approve("officer-a", zero) }},
		{"submit", approved, func(c *domain.COSCase) error { return c.MarkSubmitted(zero, "P") }},
		{"accept", func(t *testing.T) *domain.COSCase {
			c := approved(t)
			require.NoError(t, c.MarkSubmitted(at, "P"))
			return c
		}, func(c *domain.COSCase) error { return c.MarkAccepted(zero) }},
		{"reject", func(t *testing.T) *domain.COSCase {
			c := approved(t)
			require.NoError(t, c.MarkSubmitted(at, "P"))
			return c
		}, func(c *domain.COSCase) error { return c.MarkRejected(zero, "r") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := tc.mk(t)
			st, v, n := c.Status(), c.Version(), len(c.Approvers())
			require.ErrorIs(t, tc.do(c), domain.ErrInvalidInput)
			assert.Equal(t, st, c.Status())
			assert.Equal(t, v, c.Version())
			assert.Equal(t, n, len(c.Approvers()))
		})
	}
}
