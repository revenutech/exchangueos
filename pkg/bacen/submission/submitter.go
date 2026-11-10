// Package submission defines the contract for submitting regulatory reports
// to the BACEN gateway (DEC under Lei 14.286/2021, SCE-IED, SCE-Crédito and
// SCE-CBE) and polling their processing status.
package submission

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

var (
	// ErrInvalidPayload is returned when a Payload fails Validate.
	ErrInvalidPayload = errors.New("submission: invalid payload")
	// ErrProtocolNotFound is returned by Query for an unknown protocol ID.
	ErrProtocolNotFound = errors.New("submission: protocol not found")
)

// Kind is the BACEN report a payload carries.
type Kind string

const (
	KindDEC        Kind = "DEC"
	KindSCEIED     Kind = "SCE-IED"
	KindSCECredito Kind = "SCE-CREDITO"
	KindSCECBE     Kind = "SCE-CBE"
)

func (k Kind) valid() bool {
	switch k {
	case KindDEC, KindSCEIED, KindSCECredito, KindSCECBE:
		return true
	}
	return false
}

// Payload is a report already marshalled to the BACEN XML schema.
type Payload struct {
	Kind Kind
	// ReferenceID is our idempotency key for the report (e.g. the BACENReport ID).
	ReferenceID string
	// Body is the signed XML document, sent byte for byte: implementations must
	// not reformat or re-encode it, or the XAdES signature breaks. It carries
	// personal data and never goes to logs.
	Body []byte
}

// String never includes Body. fmt uses it for %v, %+v and %s.
func (p Payload) String() string {
	return fmt.Sprintf("Payload{Kind:%s ReferenceID:%s body_len:%d}", p.Kind, p.ReferenceID, len(p.Body))
}

// LogValue never includes Body.
func (p Payload) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("kind", string(p.Kind)),
		slog.String("reference_id", p.ReferenceID),
		slog.Int("body_len", len(p.Body)),
	)
}

// Validate reports whether the payload can be submitted.
func (p Payload) Validate() error {
	if !p.Kind.valid() {
		return fmt.Errorf("%w: kind %q unsupported", ErrInvalidPayload, p.Kind)
	}
	if strings.TrimSpace(p.ReferenceID) == "" {
		return fmt.Errorf("%w: reference id is required", ErrInvalidPayload)
	}
	if len(p.Body) == 0 {
		return fmt.Errorf("%w: body is empty", ErrInvalidPayload)
	}
	return nil
}

// Status is the processing state BACEN reports for a protocol.
type Status string

const (
	StatusReceived   Status = "RECEIVED"
	StatusProcessing Status = "PROCESSING"
	StatusAccepted   Status = "ACCEPTED"
	StatusRejected   Status = "REJECTED"
)

// Terminal reports whether BACEN will not change the status any more.
func (s Status) Terminal() bool { return s == StatusAccepted || s == StatusRejected }

// Receipt is BACEN's answer to a submission or a status query.
type Receipt struct {
	ProtocolID string
	Status     Status
	// RejectionReason is set only when Status is StatusRejected. It is BACEN's
	// text and may quote personal data from the XML (CPF, declarant name):
	// never log it raw, and persist it as PII.
	RejectionReason string
	ReceivedAt      time.Time
}

// Submitter sends reports to BACEN and polls their status.
type Submitter interface {
	// Submit sends p and returns the protocol BACEN assigned to it. It returns
	// ErrInvalidPayload when p.Validate fails, without calling BACEN. Submit is
	// idempotent on p.ReferenceID: a second call with a reference already
	// submitted returns the existing receipt and does not send the report again,
	// so a retry never files a duplicate declaration.
	Submit(ctx context.Context, p Payload) (Receipt, error)
	// Query returns the current status of protocolID, or ErrProtocolNotFound.
	Query(ctx context.Context, protocolID string) (Receipt, error)
}
