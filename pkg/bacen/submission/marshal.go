package submission

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"time"
	_ "time/tzdata" // embedded zone database: the host may lack it or run in another zone
	"unicode/utf8"

	"github.com/google/uuid"
)

// ErrInvalidReport is returned when a Report or an XML document fails validation.
var ErrInvalidReport = errors.New("submission: invalid report")

// ReportType mirrors modules/compliance/domain.ReportType. pkg/ never imports
// modules/, so the application layer copies the aggregate's accessors into a
// Report. The values are the domain's, unchanged.
//
// Open question (MS-024f 1.10): these are the domain report types, not the
// BACEN declaration kinds (Kind: DEC, SCE-*). This file does not map one to the
// other and builds no Payload.
type ReportType string

const (
	ReportSISBACEN ReportType = "SISBACEN"
	ReportCCS      ReportType = "BCB-CCS"
	ReportCambio   ReportType = "BCB-CAMBIO"
)

// ReportStatus mirrors modules/compliance/domain.ReportStatus.
type ReportStatus string

const (
	ReportPending   ReportStatus = "PENDING"
	ReportSubmitted ReportStatus = "SUBMITTED"
	ReportAccepted  ReportStatus = "ACCEPTED"
	ReportRejected  ReportStatus = "REJECTED"
)

// CivilDate is a calendar date with no time and no zone: the reference date of
// a report to BCB, which is the civil date in Brasília (America/Sao_Paulo).
// It is resolved at the point where the Report is built, because the domain
// BACENReport carries it two ways:
//   - a fresh report holds an instant from the proto Timestamp
//     (modules/compliance/api/grpc_server.go): use SaoPauloDateOf;
//   - a loaded report holds the DATE column read as midnight UTC
//     (modules/compliance/infrastructure/postgres/repos.go): use DateOf.
//
// Using the wrong one moves the date by one day.
type CivilDate struct {
	Year  int
	Month time.Month
	Day   int
}

// saoPaulo is the zone of BCB's civil date. America/Sao_Paulo, not a fixed
// -03:00, so that historical daylight-saving rules (until 2019) still hold.
// time/tzdata is embedded, so the lookup cannot fail on any host; a nil
// location would make SaoPauloDateOf panic instead of shifting a date.
var saoPaulo, _ = time.LoadLocation("America/Sao_Paulo")

// SaoPauloDateOf is the civil date in São Paulo at instant t, whatever t's zone.
func SaoPauloDateOf(t time.Time) CivilDate { return DateOf(t.In(saoPaulo)) }

// DateOf is the year, month and day of t in t's own zone, with no conversion:
// the right reading of a DATE column, which pgx returns as midnight UTC.
func DateOf(t time.Time) CivilDate {
	y, m, d := t.Date()
	return CivilDate{Year: y, Month: m, Day: d}
}

// String is the date as YYYY-MM-DD.
func (d CivilDate) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.Year, int(d.Month), d.Day)
}

func (d CivilDate) valid() bool {
	return d.Year >= 1 && DateOf(time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC)) == d
}

// Report is the input of MarshalReport: a copy of a domain BACENReport.
// Instants are written in UTC; ReferenceDate is a civil date (see CivilDate).
type Report struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Type          ReportType
	ReferenceDate CivilDate
	// PayloadHash is the lower-case hex SHA-256 of the serialised payload.
	PayloadHash string
	Status      ReportStatus
	SubmittedAt time.Time
	RespondedAt time.Time
	// RejectionReason is BACEN's text and may quote personal data: it is
	// never printed by String, GoString or LogValue.
	RejectionReason string
	Version         int
}

// String never includes RejectionReason. fmt uses it for %v, %+v and %s.
func (r Report) String() string {
	return fmt.Sprintf("Report{ID:%s TenantID:%s Type:%s Status:%s Version:%d rejection_reason_len:%d}",
		r.ID, r.TenantID, r.Type, r.Status, r.Version, len(r.RejectionReason))
}

// GoString never includes RejectionReason. fmt uses it for %#v.
func (r Report) GoString() string { return "submission." + r.String() }

// LogValue never includes RejectionReason.
func (r Report) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("id", r.ID.String()),
		slog.String("tenant_id", r.TenantID.String()),
		slog.String("type", string(r.Type)),
		slog.String("status", string(r.Status)),
		slog.Int("version", r.Version),
		slog.Int("rejection_reason_len", len(r.RejectionReason)),
	)
}

const (
	dateLayout = "2006-01-02"
	timeLayout = time.RFC3339Nano
)

// reportDoc is the XML shape. The element order is the field order, so the
// output is deterministic. No namespace: this is not a BACEN or ISO schema.
type reportDoc struct {
	XMLName         xml.Name `xml:"BACENReport"`
	ID              string   `xml:"ID"`
	TenantID        string   `xml:"TenantID"`
	ReportType      string   `xml:"ReportType"`
	ReferenceDate   string   `xml:"ReferenceDate"`
	PayloadHash     string   `xml:"PayloadHash"`
	Status          string   `xml:"Status"`
	SubmittedAt     string   `xml:"SubmittedAt,omitempty"`
	RespondedAt     string   `xml:"RespondedAt,omitempty"`
	RejectionReason string   `xml:"RejectionReason,omitempty"`
	Version         int      `xml:"Version"`
}

// MarshalReport validates r and encodes it as an XML document (header plus
// one element, no indentation). It never logs the content.
func MarshalReport(r Report) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	doc := reportDoc{
		ID:              r.ID.String(),
		TenantID:        r.TenantID.String(),
		ReportType:      string(r.Type),
		ReferenceDate:   r.ReferenceDate.String(),
		PayloadHash:     r.PayloadHash,
		Status:          string(r.Status),
		SubmittedAt:     formatTime(r.SubmittedAt),
		RespondedAt:     formatTime(r.RespondedAt),
		RejectionReason: r.RejectionReason,
		Version:         r.Version,
	}
	// Cannot fail: reportDoc has only string and int fields, and validate
	// already refused text that XML cannot carry.
	body, _ := xml.Marshal(doc)
	return append([]byte(xml.Header), body...), nil
}

// UnmarshalReport parses a document produced by MarshalReport and validates it.
func UnmarshalReport(b []byte) (Report, error) {
	var doc reportDoc
	if err := xml.Unmarshal(b, &doc); err != nil {
		return Report{}, fmt.Errorf("%w: decode: %v", ErrInvalidReport, err)
	}
	var (
		r   Report
		err error
	)
	if r.ID, err = uuid.Parse(doc.ID); err != nil {
		return Report{}, fmt.Errorf("%w: id: %v", ErrInvalidReport, err)
	}
	if r.TenantID, err = uuid.Parse(doc.TenantID); err != nil {
		return Report{}, fmt.Errorf("%w: tenant_id: %v", ErrInvalidReport, err)
	}
	day, err := time.Parse(dateLayout, doc.ReferenceDate)
	if err != nil {
		return Report{}, fmt.Errorf("%w: reference_date: %v", ErrInvalidReport, err)
	}
	r.ReferenceDate = DateOf(day)
	if r.SubmittedAt, err = parseTime(doc.SubmittedAt); err != nil {
		return Report{}, fmt.Errorf("%w: submitted_at: %v", ErrInvalidReport, err)
	}
	if r.RespondedAt, err = parseTime(doc.RespondedAt); err != nil {
		return Report{}, fmt.Errorf("%w: responded_at: %v", ErrInvalidReport, err)
	}
	r.Type = ReportType(doc.ReportType)
	r.PayloadHash = doc.PayloadHash
	r.Status = ReportStatus(doc.Status)
	r.RejectionReason = doc.RejectionReason
	r.Version = doc.Version
	if err := r.validate(); err != nil {
		return Report{}, err
	}
	return r, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeLayout)
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(timeLayout, s)
	return t.UTC(), err
}

func (r Report) validate() error {
	bad := func(format string, a ...any) error {
		return fmt.Errorf("%w: "+format, append([]any{ErrInvalidReport}, a...)...)
	}
	switch {
	case r.ID == uuid.Nil:
		return bad("id required")
	case r.TenantID == uuid.Nil:
		return bad("tenant_id required")
	case !r.ReferenceDate.valid():
		return bad("reference_date must be a real calendar date")
	case !isLowerHex64(r.PayloadHash):
		return bad("payload_hash must be 64 lower-case hex characters")
	case r.Version < 1:
		return bad("version must be >= 1")
	}
	switch r.Type {
	case ReportSISBACEN, ReportCCS, ReportCambio:
	default:
		return bad("report_type %q unsupported", r.Type)
	}
	submitted, responded := !r.SubmittedAt.IsZero(), !r.RespondedAt.IsZero()
	switch r.Status {
	case ReportPending:
		if submitted || responded {
			return bad("PENDING has no submitted_at or responded_at")
		}
	case ReportSubmitted:
		if !submitted || responded {
			return bad("SUBMITTED needs submitted_at and no responded_at")
		}
	case ReportAccepted, ReportRejected:
		if !submitted || !responded {
			return bad("%s needs submitted_at and responded_at", r.Status)
		}
	default:
		return bad("status %q unsupported", r.Status)
	}
	if (r.Status == ReportRejected) != (r.RejectionReason != "") {
		return bad("rejection_reason is required for REJECTED and only for it")
	}
	if !isXMLText(r.RejectionReason) {
		// encoding/xml would write U+FFFD in place of these, changing the text.
		return bad("rejection_reason has characters XML cannot carry")
	}
	return nil
}

// isXMLText reports whether s is valid UTF-8 made only of XML 1.0 Char.
func isXMLText(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		ok := c == 0x09 || c == 0x0A || c == 0x0D ||
			(c >= 0x20 && c <= 0xD7FF) || (c >= 0xE000 && c <= 0xFFFD) || (c >= 0x10000 && c <= 0x10FFFF)
		if !ok {
			return false
		}
	}
	return true
}

func isLowerHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
