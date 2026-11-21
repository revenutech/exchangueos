package fxtr

import "encoding/xml"

// Namespace015 — pinned schema namespace for fxtr.015.001.05.
const Namespace015 = "urn:iso:std:iso:20022:tech:xsd:fxtr.015.001.05"

// FXTradeAmendmentConfirmationV05 — fxtr.015.001.05.
// CLS variant: confirms amendment of a previously captured trade (RN_FX_013 — amendment > USD 100k
// requires 4-eyes; that gate runs in the application layer before this message is emitted).
type FXTradeAmendmentConfirmationV05 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.015.001.05 FXTradAmdmntConf"`

	// Original trade reference (links back to the fxtr.014 that established the trade).
	OriginalTradeIdentification TradeIdentification14 `xml:"OrgnlTradId"`

	// Amendment details — fresh side IDs + fresh economics replacing prior values.
	TradingSideIdentification      SideIdentification `xml:"TradgSdIdr"`
	CounterpartySideIdentification SideIdentification `xml:"CtrPtySdIdr"`

	AmendedTradeDate  ISODate       `xml:"AmddTradDt"`
	AmendedValueDate  ISODate       `xml:"AmddValDt"`
	AmendedTradedAmts TradedAmounts `xml:"AmddTraddAmts"`
	AmendedAgreedRate AgreedRate    `xml:"AmddAgrdRate"`

	// Amendment metadata.
	AmendmentReason string      `xml:"AmdmntRsn,omitempty"`
	AmendedAt       ISODateTime `xml:"AmdmntDtTm"`

	// Optional audit / 4-eyes (4-eyes mandatory above USD 100k — RN_FX_013).
	AuditTrail *AuditTrail `xml:"AdtTrl,omitempty"`

	// Optional updated settlement instructions.
	SettlementInfo *SettlementInfo `xml:"SttlmInfo,omitempty"`
}
