package fxtr_test

import (
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/revenutech/exchangeos/pkg/iso20022/fxtr"
	"github.com/revenutech/exchangeos/pkg/iso20022/marshaller"
	"github.com/revenutech/exchangeos/pkg/iso20022/registry"
)

// xsdFxtr013Path points to the downloaded ISO 20022 XSD for fxtr.013.001.03.
// The directory is gitignored (`.cache/`), so CI without `scripts/download-xsd.sh`
// will not find it. The test skips with an explicit message in that case —
// a silent pass would be a false green.
const xsdFxtr013Path = "../../../.cache/xsd/fxtr.013.001.03.xsd"

// readXSDOrSkip loads the fxtr.013.001.03 XSD if present, otherwise skips the test.
func readXSDOrSkip(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(xsdFxtr013Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Skipf("skipping: XSD not found at %s — run `bash scripts/download-xsd.sh` to populate .cache/xsd/",
				xsdFxtr013Path)
		}
		t.Fatalf("read XSD %s: %v", xsdFxtr013Path, err)
	}
	return raw
}

// xsdSchemaHeader captures the schema element's targetNamespace attribute,
// the smallest well-formedness check that proves this is the right XSD.
type xsdSchemaHeader struct {
	XMLName         xml.Name `xml:"http://www.w3.org/2001/XMLSchema schema"`
	TargetNamespace string   `xml:"targetNamespace,attr"`
}

func TestRoundTrip_Fxtr013_ViaMarshaller(t *testing.T) {
	xsdRaw := readXSDOrSkip(t)

	// Minimal XSD sanity: the file is well-formed XML with the expected targetNamespace.
	var header xsdSchemaHeader
	if err := xml.Unmarshal(xsdRaw, &header); err != nil {
		t.Fatalf("parse XSD: %v", err)
	}
	if got, want := header.TargetNamespace, fxtr.Namespace013; got != want {
		t.Fatalf("targetNamespace mismatch: got %q want %q", got, want)
	}

	// The XSD must declare the complex type and the five sequence elements our
	// Go struct projects. Looking for the string anchors keeps the check honest
	// without pulling in an XSD parser.
	xsdStr := string(xsdRaw)
	for _, want := range []string{
		`<xs:complexType name="ForeignExchangeTradeWithdrawalNotificationV03">`,
		`name="FXTradWdrwlNtfctn"`,
		`name="MsgId"`,
		`name="MtchgSysUnqRef"`,
		`name="WdrwlRsn"`,
		`name="SttlmSsnIdr"`,
		`name="SplmtryData"`,
		`<xs:enumeration value="RJCT"/>`,
		`<xs:enumeration value="RSCD"/>`,
		`<xs:enumeration value="WTDN"/>`,
	} {
		if !strings.Contains(xsdStr, want) {
			t.Errorf("XSD missing expected anchor %q — the struct and the XSD have drifted", want)
		}
	}

	// Round-trip through the marshaller (BAH envelope + Document wrapping).
	reg := registry.Default()
	desc, ok := reg.LookupByURN(fxtr.Namespace013)
	if !ok {
		t.Fatalf("registry lookup miss for %s", fxtr.Namespace013)
	}

	in := fxtr.ForeignExchangeTradeWithdrawalNotificationV03{
		MessageID:            "MSG-WDRWL-2026-0001",
		MatchingSysUniqueRef: "CLS-MATCH-7F2A3B9C",
		WithdrawalReason: &fxtr.WithdrawalReason1{
			Code:    "WTDN",
			SubCode: "VOID",
		},
		SettlementSessionID: "SES1",
		SupplementaryData: []fxtr.SupplementaryData1{
			{
				PlaceAndName: "CLSBUS33",
				Envelope: fxtr.SupplementaryDataEnvelope1{
					InnerXML: `<Ext xmlns="urn:revenu:ext:v1"><Note>recalled by member</Note></Ext>`,
				},
			},
		},
	}

	bah := marshaller.BAH{
		From:      "DEUTDEFF",
		To:        "CLSBUS33",
		BizMsgIdr: "BIZ-WDRWL-2026-0001",
		CreDt:     "2026-05-24T14:00:00Z",
	}

	raw, err := marshaller.Marshal(desc, bah, in, marshaller.MarshalOptions{Indent: "  "})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	xmlStr := string(raw)

	// Envelope must carry the right MsgDefIdr and the required body anchors.
	for _, want := range []string{
		"<?xml",
		"BusinessMessage",
		"AppHdr",
		"BizMsgIdr",
		"DEUTDEFF",
		"CLSBUS33",
		"MsgDefIdr",
		"fxtr.013.001.03",
		"FXTradWdrwlNtfctn",
		"MSG-WDRWL-2026-0001",
		"CLS-MATCH-7F2A3B9C",
		"WdrwlRsn",
		"WTDN",
		"SttlmSsnIdr",
		"SES1",
		"SplmtryData",
		"recalled by member",
	} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("marshaled envelope missing %q\n--- xml ---\n%s", want, xmlStr)
		}
	}

	// Round-trip via marshaller.Unmarshal.
	var out fxtr.ForeignExchangeTradeWithdrawalNotificationV03
	gotDesc, gotHdr, err := marshaller.Unmarshal(reg, raw, &out)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if gotDesc.URN() != desc.URN() {
		t.Errorf("descriptor URN: got %s want %s", gotDesc.URN(), desc.URN())
	}
	if gotHdr.BizMsgIdr != bah.BizMsgIdr {
		t.Errorf("BizMsgIdr round-trip: got %s want %s", gotHdr.BizMsgIdr, bah.BizMsgIdr)
	}
	if got, want := gotHdr.MsgDefIdr, "fxtr.013.001.03"; got != want {
		t.Errorf("MsgDefIdr: got %s want %s", got, want)
	}

	if out.MessageID != in.MessageID {
		t.Errorf("MessageID: got %q want %q", out.MessageID, in.MessageID)
	}
	if out.MatchingSysUniqueRef != in.MatchingSysUniqueRef {
		t.Errorf("MatchingSysUniqueRef: got %q want %q", out.MatchingSysUniqueRef, in.MatchingSysUniqueRef)
	}
	if out.WithdrawalReason == nil {
		t.Fatalf("WithdrawalReason: got nil, want non-nil")
	}
	if out.WithdrawalReason.Code != in.WithdrawalReason.Code {
		t.Errorf("WithdrawalReason.Code: got %q want %q", out.WithdrawalReason.Code, in.WithdrawalReason.Code)
	}
	if out.WithdrawalReason.SubCode != in.WithdrawalReason.SubCode {
		t.Errorf("WithdrawalReason.SubCode: got %q want %q", out.WithdrawalReason.SubCode, in.WithdrawalReason.SubCode)
	}
	if out.SettlementSessionID != in.SettlementSessionID {
		t.Errorf("SettlementSessionID: got %q want %q", out.SettlementSessionID, in.SettlementSessionID)
	}
	if n := len(out.SupplementaryData); n != 1 {
		t.Fatalf("SupplementaryData length: got %d want 1", n)
	}
	if got, want := out.SupplementaryData[0].PlaceAndName, "CLSBUS33"; got != want {
		t.Errorf("SupplementaryData[0].PlaceAndName: got %q want %q", got, want)
	}
	if !strings.Contains(out.SupplementaryData[0].Envelope.InnerXML, "recalled by member") {
		t.Errorf("SupplementaryData[0].Envelope.InnerXML lost payload: %q",
			out.SupplementaryData[0].Envelope.InnerXML)
	}
}
