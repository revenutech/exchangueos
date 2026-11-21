package fxtr

import "encoding/xml"

// Namespace008 — pinned schema namespace for fxtr.008.001.07.
const Namespace008 = "urn:iso:std:iso:20022:tech:xsd:fxtr.008.001.07"

// FXTradeNotificationV07 — fxtr.008.001.07.
// CLS variant: emitted by CLS to a settlement member acknowledging trade
// receipt into the daily settlement queue. Pre-cursor to fxtr.014 confirmation.
type FXTradeNotificationV07 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.008.001.07 FXTradNtfctn"`

	// Echoed identification block from the original capture.
	TradeIdentification TradeIdentification14 `xml:"TradId"`

	// Sides — populated for audit even though CLS already knows them.
	TradingSideIdentification      SideIdentification `xml:"TradgSdIdr"`
	CounterpartySideIdentification SideIdentification `xml:"CtrPtySdIdr"`

	// Settlement queue context.
	QueuedAt       ISODateTime `xml:"QudDtTm"`
	CLSCycleID     string      `xml:"ClsCyclId,omitempty"` // YYYY-MM-DD-NNN
	SettlementDate ISODate     `xml:"SttlmDt"`

	// Optional reason if the notification carries a status (otherwise SUCCESS implicit).
	StatusReason string `xml:"StsRsn,omitempty"`
}
