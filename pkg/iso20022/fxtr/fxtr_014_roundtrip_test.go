package fxtr_test

import (
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/iso20022/fxtr"
	"github.com/revenutech/exchangeos/pkg/iso20022/marshaller"
	"github.com/revenutech/exchangeos/pkg/iso20022/registry"
)

func sampleFxtr014() fxtr.FXTradeCaptureConfirmationV05 {
	// Midnight, not 14:00. TradDt and ValDt are ISODate (ISO 20022 date, no
	// time of day), so the marshaller correctly drops any time component —
	// seeding one made the round-trip comparison fail against the marshaller
	// doing the right thing.
	tradeDate := time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC)
	return fxtr.FXTradeCaptureConfirmationV05{
		TradeIdentification: fxtr.TradeIdentification14{
			TradeID:         "T-2026-0001",
			OurTradeRef:     "OURREF-001",
			CommonReference: "CLS-CMN-9999",
			CapturedAt:      fxtr.ISODateTime{Time: tradeDate},
		},
		TradingSideIdentification: fxtr.SideIdentification{
			SubmitterID: fxtr.PartyIdentification{BICFI: "DEUTDEFF", LEI: "7LTWFZYICNSX8D621K86", Name: "Deutsche Bank AG"},
		},
		CounterpartySideIdentification: fxtr.SideIdentification{
			SubmitterID: fxtr.PartyIdentification{BICFI: "CHASUS33", Name: "JPMorgan Chase"},
		},
		TradeDate: fxtr.ISODate{Time: tradeDate},
		ValueDate: fxtr.ISODate{Time: tradeDate.Add(48 * time.Hour)},
		TradedAmts: fxtr.TradedAmounts{
			BoughtAmount: fxtr.Amount{Currency: "EUR", Value: decimal.NewFromInt(1_000_000)},
			SoldAmount:   fxtr.Amount{Currency: "USD", Value: decimal.RequireFromString("1080123.456789")},
		},
		AgreedRate: fxtr.AgreedRate{
			ExchangeRate:   fxtr.Rate{Value: decimal.RequireFromString("1.080123456789")},
			BaseCurrency:   "EUR",
			QuotedCurrency: "USD",
		},
	}
}

func TestRoundTrip_Fxtr014_ViaMarshaller(t *testing.T) {
	reg := registry.Default()
	desc, ok := reg.LookupByURN(fxtr.Namespace014)
	if !ok {
		t.Fatalf("registry lookup miss for %s", fxtr.Namespace014)
	}

	in := sampleFxtr014()
	header := marshaller.BAH{
		From:      "DEUTDEFF",
		To:        "CLSBUS33",
		BizMsgIdr: "BIZ-T-2026-0001",
		CreDt:     "2026-05-24T14:00:00Z",
	}

	raw, err := marshaller.Marshal(desc, header, in, marshaller.MarshalOptions{Indent: "  "})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	xmlStr := string(raw)

	// Sanity checks on the envelope.
	for _, want := range []string{
		"<?xml",
		"BusinessMessage",
		"AppHdr",
		"BizMsgIdr",
		"DEUTDEFF",
		"CLSBUS33",
		"MsgDefIdr",
		"fxtr.014.001.05",
		"FXTradCaptrConf",
		"1.080123456789",
		"1080123.456789",
		"EUR",
		"USD",
		"2026-05-24",
	} {
		if !strings.Contains(xmlStr, want) {
			t.Errorf("envelope missing %q\n--- xml ---\n%s", want, xmlStr)
		}
	}

	// Round-trip via marshaller.Unmarshal — populates BAH + body.
	var out fxtr.FXTradeCaptureConfirmationV05
	gotDesc, gotHdr, err := marshaller.Unmarshal(reg, raw, &out)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if gotDesc.URN() != desc.URN() {
		t.Errorf("descriptor URN: got %s want %s", gotDesc.URN(), desc.URN())
	}
	if gotHdr.BizMsgIdr != header.BizMsgIdr {
		t.Errorf("BizMsgIdr round-trip: got %s want %s", gotHdr.BizMsgIdr, header.BizMsgIdr)
	}
	if got, want := gotHdr.MsgDefIdr, "fxtr.014.001.05"; got != want {
		t.Errorf("MsgDefIdr: got %s want %s", got, want)
	}

	// Body equality checks (don't require full struct equality — focus on the dealt values).
	if out.TradeIdentification.TradeID != in.TradeIdentification.TradeID {
		t.Errorf("TradeID: got %s want %s", out.TradeIdentification.TradeID, in.TradeIdentification.TradeID)
	}
	if !out.TradedAmts.BoughtAmount.Value.Equal(in.TradedAmts.BoughtAmount.Value) {
		t.Errorf("BoughtAmount: got %s want %s",
			out.TradedAmts.BoughtAmount.Value, in.TradedAmts.BoughtAmount.Value)
	}
	if !out.TradedAmts.SoldAmount.Value.Equal(in.TradedAmts.SoldAmount.Value) {
		t.Errorf("SoldAmount: got %s want %s",
			out.TradedAmts.SoldAmount.Value, in.TradedAmts.SoldAmount.Value)
	}
	if !out.AgreedRate.ExchangeRate.Value.Equal(in.AgreedRate.ExchangeRate.Value) {
		t.Errorf("Rate: got %s want %s",
			out.AgreedRate.ExchangeRate.Value, in.AgreedRate.ExchangeRate.Value)
	}
	if !out.TradeDate.Time.Equal(in.TradeDate.Time) {
		t.Errorf("TradeDate: got %v want %v", out.TradeDate.Time, in.TradeDate.Time)
	}
	if !out.ValueDate.Time.Equal(in.ValueDate.Time) {
		t.Errorf("ValueDate: got %v want %v", out.ValueDate.Time, in.ValueDate.Time)
	}
	if out.TradingSideIdentification.SubmitterID.BICFI != "DEUTDEFF" {
		t.Errorf("Trading-side BICFI: got %s want DEUTDEFF",
			out.TradingSideIdentification.SubmitterID.BICFI)
	}
}

func TestUnmarshal_UnknownURN_ReturnsError(t *testing.T) {
	reg := registry.Default()
	bogus := []byte(`<?xml version="1.0"?>
<BusinessMessage xmlns="urn:iso:std:iso:20022:tech:xsd:head.001.001.03">
  <AppHdr>
    <Fr><FIId><FinInstnId><BICFI>DEUTDEFF</BICFI></FinInstnId></FIId></Fr>
    <To><FIId><FinInstnId><BICFI>CLSBUS33</BICFI></FinInstnId></FIId></To>
    <BizMsgIdr>BIZ-X</BizMsgIdr>
    <MsgDefIdr>fxtr.999.001.99</MsgDefIdr>
    <CreDt>2026-05-24T14:00:00Z</CreDt>
  </AppHdr>
  <Body/>
</BusinessMessage>`)
	_, _, err := marshaller.Unmarshal(reg, bogus, nil)
	if err == nil || !strings.Contains(err.Error(), "unknown URN") {
		t.Fatalf("expected unknown-URN error, got %v", err)
	}
}
