package camt_test

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/iso20022/camt"
)

// xsdCamt088V03Path points to the ISO 20022 XSD for camt.088.001.03. The
// `.cache/xsd/` directory is gitignored (populated by scripts/download-xsd.sh),
// so CI without the download step will not find it. Test skips with an
// explicit message in that case — a silent pass would be a false green.
const xsdCamt088V03Path = "../../../.cache/xsd/camt.088.001.03.xsd"

func readCamt088V03XSDOrSkip(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(xsdCamt088V03Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("skipping: XSD not found at %s — run `bash scripts/download-xsd.sh` to populate .cache/xsd/",
				xsdCamt088V03Path)
		}
		t.Fatalf("read XSD %s: %v", xsdCamt088V03Path, err)
	}
	return raw
}

type xsdSchemaHeader struct {
	XMLName         xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	TargetNamespace string   `xml:"targetNamespace,attr"`
}

func TestRoundTrip_Camt088V03_SchemaAwareMarshalUnmarshal(t *testing.T) {
	xsdRaw := readCamt088V03XSDOrSkip(t)

	// The XSD is well-formed XML with the expected targetNamespace.
	var header xsdSchemaHeader
	if err := xml.Unmarshal(xsdRaw, &header); err != nil {
		t.Fatalf("parse XSD: %v", err)
	}
	if got, want := header.TargetNamespace, camt.Namespace088V03; got != want {
		t.Fatalf("targetNamespace mismatch: got %q want %q", got, want)
	}

	// Anchor every element name the Go projection reads/writes.
	xsdStr := string(xsdRaw)
	for _, want := range []string{
		`<xs:complexType name="NetReportV03">`,
		`<xs:complexType name="NetReportData2">`,
		`<xs:complexType name="NetObligation3">`,
		`name="NetRpt"`,
		`name="NetRptData"`,
		`name="NetSvcPtcptId"`,
		`name="NetSvcCtrPtyId"`,
		`name="NetOblgtn"`,
		`name="MsgId"`,
		`name="CreDtTm"`,
		`name="NetgCutOffTm"`,
		`name="RptDt"`,
		`name="ValDt"`,
		`name="OblgtnId"`,
		`name="Amt"`,
		`name="PtcptNetgId"`,
		`name="OblgtnDrctn"`,
		`name="CtrPtyNetgId"`,
		`<xs:enumeration value="PAYM"/>`,
		`<xs:enumeration value="RECE"/>`,
		`<xs:enumeration value="NONE"/>`,
		`name="AnyBIC"`,
		`name="TradPty"`,
	} {
		if !strings.Contains(xsdStr, want) {
			t.Errorf("XSD missing expected anchor %q — the struct and the XSD have drifted", want)
		}
	}

	// Build a sample report with two obligations in opposite directions.
	memberBIC := "DEUTDEFF"
	clsBIC := "CLSBUS33"

	in := camt.NetReportV03{
		ReportData: camt.NetReportData2{
			MessageID:         "NR-2026-05-26-001",
			CreatedAt:         camt.ISODateTime{Value: "2026-05-26T10:15:00Z"},
			NettingCutOffTime: camt.ISOTime{Value: "10:00:00"},
			ReportDate:        camt.ISODate{Value: "2026-05-26"},
			ValueDate:         camt.ISODate{Value: "2026-05-26"},
			ReportType:        "FINAL",
		},
		NetServiceParticipant: camt.PartyIdentification242Choice{
			AnyBIC: &camt.PartyIdentification265{AnyBIC: memberBIC},
		},
		NetServiceCounterparty: &camt.PartyIdentification242Choice{
			AnyBIC: &camt.PartyIdentification265{AnyBIC: clsBIC},
		},
		Obligations: []camt.NetObligation3{
			{
				ObligationID: "OB-EUR-001",
				Amount: camt.Amount{
					Currency: "EUR",
					// shopspring/decimal.String() strips trailing zeros, so
					// 250000.00 marshals as "250000". ISO 20022
					// ActiveCurrencyAndAmount does not require trailing zero
					// padding at the type level; CLS SR2026 does not
					// contradict that here.
					Value: decimal.RequireFromString("250000"),
				},
				ParticipantNettingID: camt.NettingIdentification2Choice{
					TradingParty: &camt.PartyIdentification242Choice{
						AnyBIC: &camt.PartyIdentification265{AnyBIC: memberBIC},
					},
				},
				ObligationDirection: "RECE",
				CounterpartyNettingID: camt.NettingIdentification2Choice{
					TradingParty: &camt.PartyIdentification242Choice{
						AnyBIC: &camt.PartyIdentification265{AnyBIC: clsBIC},
					},
				},
			},
			{
				ObligationID: "OB-USD-001",
				Amount: camt.Amount{
					Currency: "USD",
					Value:    decimal.RequireFromString("270123.45"),
				},
				ParticipantNettingID: camt.NettingIdentification2Choice{
					TradingParty: &camt.PartyIdentification242Choice{
						AnyBIC: &camt.PartyIdentification265{AnyBIC: memberBIC},
					},
				},
				ObligationDirection: "PAYM",
				CounterpartyNettingID: camt.NettingIdentification2Choice{
					TradingParty: &camt.PartyIdentification242Choice{
						AnyBIC: &camt.PartyIdentification265{AnyBIC: clsBIC},
					},
				},
			},
		},
	}

	raw, err := xml.MarshalIndent(in, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	xmlStr := string(raw)

	// Every anchor the XSD declares must show up in the marshaled XML.
	for _, want := range []string{
		`xmlns="urn:iso:std:iso:20022:tech:xsd:camt.088.001.03"`,
		`<NetRpt`,
		`<NetRptData>`,
		`<MsgId>NR-2026-05-26-001</MsgId>`,
		`<NetgCutOffTm>10:00:00</NetgCutOffTm>`,
		`<RptDt>2026-05-26</RptDt>`,
		`<ValDt>2026-05-26</ValDt>`,
		`<NetSvcPtcptId>`,
		`<AnyBIC>DEUTDEFF</AnyBIC>`,
		`<NetSvcCtrPtyId>`,
		`<AnyBIC>CLSBUS33</AnyBIC>`,
		`<NetOblgtn>`,
		`<OblgtnId>OB-EUR-001</OblgtnId>`,
		`<Amt Ccy="EUR">250000</Amt>`,
		`<OblgtnDrctn>RECE</OblgtnDrctn>`,
		`<Amt Ccy="USD">270123.45</Amt>`,
		`<OblgtnDrctn>PAYM</OblgtnDrctn>`,
	} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("marshaled XML missing %q\n--- xml ---\n%s", want, xmlStr)
		}
	}

	// Round-trip back to the Go struct.
	var out camt.NetReportV03
	if err := xml.Unmarshal(raw, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if out.ReportData.MessageID != in.ReportData.MessageID {
		t.Errorf("MessageID: got %q want %q", out.ReportData.MessageID, in.ReportData.MessageID)
	}
	if out.ReportData.NettingCutOffTime.Value != in.ReportData.NettingCutOffTime.Value {
		t.Errorf("NettingCutOffTime: got %q want %q",
			out.ReportData.NettingCutOffTime.Value, in.ReportData.NettingCutOffTime.Value)
	}
	if out.ReportData.ReportDate.Value != in.ReportData.ReportDate.Value {
		t.Errorf("ReportDate: got %q want %q",
			out.ReportData.ReportDate.Value, in.ReportData.ReportDate.Value)
	}
	if out.NetServiceParticipant.AnyBIC == nil ||
		out.NetServiceParticipant.AnyBIC.AnyBIC != memberBIC {
		t.Errorf("NetServiceParticipant.AnyBIC: got %+v want %q", out.NetServiceParticipant.AnyBIC, memberBIC)
	}
	if out.NetServiceCounterparty == nil || out.NetServiceCounterparty.AnyBIC == nil ||
		out.NetServiceCounterparty.AnyBIC.AnyBIC != clsBIC {
		t.Errorf("NetServiceCounterparty.AnyBIC: got %+v want %q", out.NetServiceCounterparty, clsBIC)
	}
	if n := len(out.Obligations); n != 2 {
		t.Fatalf("Obligations length: got %d want 2", n)
	}

	// Preservation of currency + sign + direction across the obligations.
	assertObligation(t, out.Obligations[0], "OB-EUR-001", "EUR", "250000", "RECE")
	assertObligation(t, out.Obligations[1], "OB-USD-001", "USD", "270123.45", "PAYM")
}

func assertObligation(t *testing.T, got camt.NetObligation3, wantID, wantCcy, wantAmt, wantDir string) {
	t.Helper()
	if got.ObligationID != wantID {
		t.Errorf("ObligationID: got %q want %q", got.ObligationID, wantID)
	}
	if got.Amount.Currency != wantCcy {
		t.Errorf("Amount.Currency: got %q want %q", got.Amount.Currency, wantCcy)
	}
	if got, want := got.Amount.Value.String(), wantAmt; got != want {
		t.Errorf("Amount.Value: got %q want %q", got, want)
	}
	if got.ObligationDirection != wantDir {
		t.Errorf("ObligationDirection: got %q want %q", got.ObligationDirection, wantDir)
	}
}
