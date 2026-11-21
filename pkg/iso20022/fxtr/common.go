package fxtr

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ─────────────────────────────────────────────────────────────────────────────
// Amount — ActiveCurrencyAndAmount / ActiveOrHistoricCurrencyAndAmount
//   <Amt Ccy="USD">1234567.89</Amt>
// ─────────────────────────────────────────────────────────────────────────────

type Amount struct {
	Currency string          `xml:"Ccy,attr"`
	Value    decimal.Decimal `xml:",chardata"`
}

func (a Amount) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Currency != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "Ccy"}, Value: a.Currency})
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	if err := e.EncodeToken(xml.CharData(a.Value.String())); err != nil {
		return err
	}
	return e.EncodeToken(start.End())
}

func (a *Amount) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "Ccy" {
			a.Currency = attr.Value
		}
	}
	var raw string
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		a.Value = decimal.Zero
		return nil
	}
	v, err := decimal.NewFromString(raw)
	if err != nil {
		return fmt.Errorf("fxtr.Amount: %w", err)
	}
	a.Value = v
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Rate — BaseOneRate / PercentageRate
//   <Rate>1.0843</Rate>
// ─────────────────────────────────────────────────────────────────────────────

type Rate struct{ Value decimal.Decimal }

func (r Rate) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(r.Value.String(), start)
}

func (r *Rate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		r.Value = decimal.Zero
		return nil
	}
	v, err := decimal.NewFromString(raw)
	if err != nil {
		return fmt.Errorf("fxtr.Rate: %w", err)
	}
	r.Value = v
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ISODate — yyyy-MM-dd
// ─────────────────────────────────────────────────────────────────────────────

type ISODate struct{ Time time.Time }

const isoDateLayout = "2006-01-02"

func (t ISODate) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(t.Time.UTC().Format(isoDateLayout), start)
}

func (t *ISODate) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	v, err := time.Parse(isoDateLayout, raw)
	if err != nil {
		return fmt.Errorf("fxtr.ISODate: %w", err)
	}
	t.Time = v
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// ISODateTime — yyyy-MM-ddTHH:mm:ss(.fff)Z
// ─────────────────────────────────────────────────────────────────────────────

type ISODateTime struct{ Time time.Time }

func (t ISODateTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(t.Time.UTC().Format(time.RFC3339Nano), start)
}

func (t *ISODateTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var raw string
	if err := d.DecodeElement(&raw, &start); err != nil {
		return err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	v, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		// fallback to non-nano variant
		v, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return fmt.Errorf("fxtr.ISODateTime: %w", err)
		}
	}
	t.Time = v
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// PartyIdentification — Trading + Counterparty sides share this shape.
// Minimal subset used by CLS: BICFI + (optional) LEI + Name.
// ─────────────────────────────────────────────────────────────────────────────

type PartyIdentification struct {
	BICFI string `xml:"FinInstnId>BICFI,omitempty"`
	LEI   string `xml:"FinInstnId>LEI,omitempty"`
	Name  string `xml:"FinInstnId>Nm,omitempty"`
}

// SideIdentification — TradgSdIdr / CtrPtySdIdr wrapper.
type SideIdentification struct {
	SubmitterID PartyIdentification `xml:"SubmitgPty"`
	TradingPart PartyIdentification `xml:"TradgPty,omitempty"`
}

// TradedAmounts — the two legs of an FX trade.
type TradedAmounts struct {
	BoughtAmount Amount `xml:"BaseProdctAmt"`
	SoldAmount   Amount `xml:"OthrProdctAmt"`
}

// AgreedRate carries the dealt FX rate + base/quote pair.
type AgreedRate struct {
	ExchangeRate   Rate   `xml:"XchgRate"`
	BaseCurrency   string `xml:"BaseCcy"`
	QuotedCurrency string `xml:"QtdCcy"`
}

// SettlementInfo — how the trade settles. CLS uses CLSStlmInfo; bilateral uses StgInstrs.
type SettlementInfo struct {
	CLSSettlement   bool   `xml:"ClrgSysAplbl,omitempty"`
	SettlementVenue string `xml:"SttlmMtd,omitempty"` // CLSS | BILA | OTHR
}

// AuditTrail — Originator + Optional 2nd actor (4-eyes RN_FX_013).
type AuditTrail struct {
	OriginatorBICFI string      `xml:"OrgtrFI,omitempty"`
	OriginatedAt    ISODateTime `xml:"OrgtnDtTm,omitempty"`
	ApproverBICFI   string      `xml:"AppnFI,omitempty"`
	ApprovedAt      ISODateTime `xml:"AppnDtTm,omitempty"`
}
