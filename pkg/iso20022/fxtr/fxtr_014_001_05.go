package fxtr

import "encoding/xml"

// Namespace014 — pinned schema namespace for fxtr.014.001.05.
const Namespace014 = "urn:iso:std:iso:20022:tech:xsd:fxtr.014.001.05"

// FXTradeCaptureConfirmationV05 — fxtr.014.001.05.
// CLS submitter variant: primary FX trade confirmation for spot/forward trades.
// Sent by either trading-side or counterparty-side member to confirm trade economics.
type FXTradeCaptureConfirmationV05 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.014.001.05 FXTradCaptrConf"`

	// Identification block — TradId is the system-wide deal identifier.
	TradeIdentification TradeIdentification14 `xml:"TradId"`

	// Side identification (CLS member or trader-of-record).
	TradingSideIdentification      SideIdentification `xml:"TradgSdIdr"`
	CounterpartySideIdentification SideIdentification `xml:"CtrPtySdIdr"`

	// Trade economics.
	TradeDate  ISODate       `xml:"TradDt"`
	ValueDate  ISODate       `xml:"ValDt"`
	TradedAmts TradedAmounts `xml:"TraddAmts"`
	AgreedRate AgreedRate    `xml:"AgrdRate"`

	// Optional: settlement instructions block.
	SettlementInfo *SettlementInfo `xml:"SttlmInfo,omitempty"`

	// Optional: audit / 4-eyes block (RN_FX_013).
	AuditTrail *AuditTrail `xml:"AdtTrl,omitempty"`
}

// TradeIdentification14 — minimal identifier block (full XSD has more optional fields).
type TradeIdentification14 struct {
	TradeID         string      `xml:"TradId"` // counterparty's id
	OurTradeRef     string      `xml:"OurTradRef,omitempty"`
	CommonReference string      `xml:"CmnRef,omitempty"` // optional CLS common reference
	CapturedAt      ISODateTime `xml:"CaptrDtTm,omitempty"`
}
