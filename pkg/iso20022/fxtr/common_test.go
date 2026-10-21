package fxtr

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestAmount_RoundTripPreservesPrecision(t *testing.T) {
	v, err := decimal.NewFromString("1234567.890123456789")
	if err != nil {
		t.Fatalf("decimal seed: %v", err)
	}
	// Amount as a named field, not embedded in an anonymous struct.
	//
	// Embedding promoted Amount.MarshalXML (value receiver) onto the wrapper,
	// so encoding/xml treated the wrapper itself as an xml.Marshaler. For an
	// unnamed type, defaultStart assumes the only way to reach that branch is a
	// pointer to a named type and calls typ.Elem(), which panics on a struct:
	//
	//   panic: reflect: Elem of invalid type struct { XMLName xml.Name ...; fxtr.Amount }
	//
	// The wrapper's own XMLName was never consulted either. A named field is
	// also how Amount actually appears inside the fxtr messages.
	in := amountEnvelope{Amt: Amount{Currency: "USD", Value: v}}

	raw, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `Ccy="USD"`) {
		t.Fatalf("missing Ccy attribute: %s", raw)
	}
	if !strings.Contains(string(raw), "1234567.890123456789") {
		t.Fatalf("decimal precision lost: %s", raw)
	}
	if !strings.Contains(string(raw), "<Amt") {
		t.Fatalf("element is not named Amt: %s", raw)
	}

	var out amountEnvelope
	if err := xml.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Amt.Currency != "USD" {
		t.Fatalf("Ccy: got %s want USD", out.Amt.Currency)
	}
	if !out.Amt.Value.Equal(v) {
		t.Fatalf("Value: got %s want %s", out.Amt.Value, v)
	}
}

// amountEnvelope mirrors how Amount is carried inside an fxtr message.
type amountEnvelope struct {
	XMLName xml.Name `xml:"Doc"`
	Amt     Amount   `xml:"Amt"`
}

func TestRate_RoundTrip(t *testing.T) {
	// Named field rather than anonymous embedding — same reason as
	// TestAmount_RoundTripPreservesPrecision above. This failure was masked
	// until now: the Amount panic aborted the whole test binary before this
	// test ever ran.
	in := rateEnvelope{Rate: Rate{Value: decimal.RequireFromString("1.087654321")}}

	raw, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), "1.087654321") {
		t.Fatalf("decimal precision lost: %s", raw)
	}

	var out rateEnvelope
	if err := xml.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Rate.Value.Equal(in.Rate.Value) {
		t.Fatalf("Rate: got %s want %s", out.Rate.Value, in.Rate.Value)
	}
}

// rateEnvelope mirrors how Rate is carried inside an fxtr message.
type rateEnvelope struct {
	XMLName xml.Name `xml:"Doc"`
	Rate    Rate     `xml:"Rate"`
}

func TestISODate_RoundTrip(t *testing.T) {
	// Named field, not anonymous embedding — see the comment on
	// TestAmount_RoundTripPreservesPrecision.
	in := isoDateEnvelope{Dt: ISODate{Time: time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)}}

	raw, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), "2026-05-24") {
		t.Fatalf("expected ISO date in output: %s", raw)
	}

	var out isoDateEnvelope
	if err := xml.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.Dt.Time.Equal(in.Dt.Time) {
		t.Fatalf("Date: got %v want %v", out.Dt.Time, in.Dt.Time)
	}
}

// isoDateEnvelope mirrors how ISODate is carried inside an fxtr message.
type isoDateEnvelope struct {
	XMLName xml.Name `xml:"Doc"`
	Dt      ISODate  `xml:"Dt"`
}

func TestISODateTime_RoundTrip(t *testing.T) {
	// Named field, not anonymous embedding — see the comment on
	// TestAmount_RoundTripPreservesPrecision.
	in := isoDateTimeEnvelope{DtTm: ISODateTime{Time: time.Date(2026, 5, 24, 14, 30, 45, 0, time.UTC)}}

	raw, err := xml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out isoDateTimeEnvelope
	if err := xml.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !out.DtTm.Time.Equal(in.DtTm.Time) {
		t.Fatalf("DateTime: got %v want %v", out.DtTm.Time, in.DtTm.Time)
	}
}

// isoDateTimeEnvelope mirrors how ISODateTime is carried inside an fxtr message.
type isoDateTimeEnvelope struct {
	XMLName xml.Name    `xml:"Doc"`
	DtTm    ISODateTime `xml:"DtTm"`
}

func TestAmount_EmptyValueDecodesAsZero(t *testing.T) {
	raw := []byte(`<Amt Ccy="EUR"></Amt>`)
	var a Amount
	if err := xml.Unmarshal(raw, &a); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !a.Value.IsZero() {
		t.Fatalf("expected zero, got %s", a.Value)
	}
	if a.Currency != "EUR" {
		t.Fatalf("expected EUR, got %s", a.Currency)
	}
}
