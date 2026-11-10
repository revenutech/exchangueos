package submission_test

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revenutech/exchangeos/pkg/bacen/submission"
)

const hashHex = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"

func validReport() submission.Report {
	return submission.Report{
		ID:            uuid.MustParse("6f1c2a8e-3b4d-4e5f-8a9b-0c1d2e3f4a5b"),
		TenantID:      uuid.MustParse("0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d"),
		Type:          submission.ReportSISBACEN,
		ReferenceDate: time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC),
		PayloadHash:   hashHex,
		Status:        submission.ReportPending,
		Version:       1,
	}
}

// One fixture per domain ReportType: parse → marshal gives back the same bytes.
func TestMarshal_RoundTripFixtures(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file string
		typ  submission.ReportType
	}{
		{"sisbacen.xml", submission.ReportSISBACEN},
		{"ccs.xml", submission.ReportCCS},
		{"cambio.xml", submission.ReportCambio},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			t.Parallel()
			want, err := os.ReadFile(filepath.Join("testdata", tc.file))
			require.NoError(t, err)
			r, err := submission.UnmarshalReport(want)
			require.NoError(t, err)
			assert.Equal(t, tc.typ, r.Type)
			got, err := submission.MarshalReport(r)
			require.NoError(t, err)
			assert.Equal(t, string(want), string(got))
		})
	}
}

func TestMarshal_ReportRoundTrip(t *testing.T) {
	t.Parallel()
	r := validReport()
	r.Type = submission.ReportCambio
	r.Status = submission.ReportRejected
	r.SubmittedAt = time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC)
	r.RespondedAt = time.Date(2026, 11, 10, 14, 0, 0, 123456789, time.UTC)
	r.RejectionReason = `campo <7> & "natureza"`
	r.Version = 3

	b, err := submission.MarshalReport(r)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(b, []byte(`<?xml version="1.0" encoding="UTF-8"?>`+"\n")))
	assert.NotContains(t, string(b), "<7>", "text must be escaped")

	back, err := submission.UnmarshalReport(b)
	require.NoError(t, err)
	assert.Equal(t, r, back)

	again, err := submission.MarshalReport(back)
	require.NoError(t, err)
	assert.Equal(t, b, again, "marshal is deterministic")
}

func TestMarshal_NormalisesTimesToUTC(t *testing.T) {
	t.Parallel()
	sp := time.FixedZone("BRT", -3*3600)
	r := validReport()
	r.ReferenceDate = time.Date(2026, 11, 9, 22, 0, 0, 0, sp) // 2026-11-10 01:00 UTC
	r.Status = submission.ReportSubmitted
	r.SubmittedAt = time.Date(2026, 11, 10, 10, 0, 0, 0, sp)
	b, err := submission.MarshalReport(r)
	require.NoError(t, err)
	assert.Contains(t, string(b), "<ReferenceDate>2026-11-10</ReferenceDate>")
	assert.Contains(t, string(b), "<SubmittedAt>2026-11-10T13:00:00Z</SubmittedAt>")
}

func TestMarshal_RejectsInvalidReport(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 10, 13, 0, 0, 0, time.UTC)
	cases := map[string]func(*submission.Report){
		"nil id":               func(r *submission.Report) { r.ID = uuid.Nil },
		"nil tenant":           func(r *submission.Report) { r.TenantID = uuid.Nil },
		"unknown type":         func(r *submission.Report) { r.Type = "DEC" },
		"zero reference date":  func(r *submission.Report) { r.ReferenceDate = time.Time{} },
		"short hash":           func(r *submission.Report) { r.PayloadHash = "abc" },
		"non-hex hash":         func(r *submission.Report) { r.PayloadHash = strings.Repeat("z", 64) },
		"upper-case hash":      func(r *submission.Report) { r.PayloadHash = strings.ToUpper(hashHex) },
		"unknown status":       func(r *submission.Report) { r.Status = "DONE" },
		"version zero":         func(r *submission.Report) { r.Version = 0 },
		"submitted no time":    func(r *submission.Report) { r.Status = submission.ReportSubmitted },
		"pending with time":    func(r *submission.Report) { r.SubmittedAt = at },
		"accepted no response": func(r *submission.Report) { r.Status = submission.ReportAccepted; r.SubmittedAt = at },
		"submitted with response": func(r *submission.Report) {
			r.Status = submission.ReportSubmitted
			r.SubmittedAt = at
			r.RespondedAt = at
		},
		"rejected no reason": func(r *submission.Report) {
			r.Status = submission.ReportRejected
			r.SubmittedAt, r.RespondedAt = at, at
		},
		"reason when not rejected": func(r *submission.Report) { r.RejectionReason = "x" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			r := validReport()
			mutate(&r)
			_, err := submission.MarshalReport(r)
			require.ErrorIs(t, err, submission.ErrInvalidReport)
		})
	}
}

func TestUnmarshal_RejectsBadDocuments(t *testing.T) {
	t.Parallel()
	good, err := os.ReadFile(filepath.Join("testdata", "ccs.xml"))
	require.NoError(t, err)
	doc := string(good)
	cases := map[string]string{
		"not xml":          "not xml",
		"wrong root":       strings.ReplaceAll(doc, "BACENReport>", "DEC>"),
		"bad id":           strings.Replace(doc, "7a2d3b9f-4c5e-4f60-9b0c-1d2e3f4a5b6c", "x", 1),
		"bad tenant":       strings.Replace(doc, "0a1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d", "x", 1),
		"bad date":         strings.Replace(doc, "<ReferenceDate>2026-11-09<", "<ReferenceDate>09/11/2026<", 1),
		"bad submitted at": strings.Replace(doc, "<SubmittedAt>2026-11-10T13:00:00Z<", "<SubmittedAt>ontem<", 1),
		"bad responded at": strings.Replace(doc, "<RespondedAt>2026-11-10T13:05:30.5Z<", "<RespondedAt>ontem<", 1),
		"invalid content":  strings.Replace(doc, "<ReportType>BCB-CCS<", "<ReportType>DEC<", 1),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := submission.UnmarshalReport([]byte(in))
			require.ErrorIs(t, err, submission.ErrInvalidReport)
		})
	}
}

// RejectionReason is BACEN's text and may quote personal data: never printed.
func TestReport_NeverLeaksRejectionReason(t *testing.T) {
	t.Parallel()
	const secret = "CPF 123.456.789-09 Joao da Silva"
	r := validReport()
	r.Status = submission.ReportRejected
	r.RejectionReason = secret

	var outs []string
	for _, f := range []string{"%v", "%+v", "%s", "%#v"} {
		outs = append(outs, fmt.Sprintf(f, r))
	}
	var text, js bytes.Buffer
	slog.New(slog.NewTextHandler(&text, nil)).Info("report", "r", r)
	slog.New(slog.NewJSONHandler(&js, nil)).Info("report", "r", r)
	outs = append(outs, text.String(), js.String())

	for i, out := range outs {
		if strings.Contains(out, "123.456.789-09") || strings.Contains(out, "Joao") {
			t.Errorf("output %d leaks the rejection reason: %s", i, out)
		}
		if !strings.Contains(out, "SISBACEN") || !strings.Contains(out, r.ID.String()) || !strings.Contains(out, fmt.Sprint(len(secret))) {
			t.Errorf("output %d lost type, id or reason length: %s", i, out)
		}
	}
}
