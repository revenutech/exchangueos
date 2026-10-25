package fxtr

import "encoding/xml"

// Namespace013 — pinned schema namespace for fxtr.013.001.03.
//
// fxtr.013 is the Foreign Exchange Trade Withdrawal Notification — a message
// sent to retire a trade from the settlement queue (recall, rejected matching,
// unilateral withdrawal). The ISO 20022 SR2026 catalogue lists .001.03 as the
// latest variant; .001.04 is not published.
const Namespace013 = "urn:iso:std:iso:20022:tech:xsd:fxtr.013.001.03"

// ForeignExchangeTradeWithdrawalNotificationV03 — fxtr.013.001.03.
//
// Fields mirror the XSD at `.cache/xsd/fxtr.013.001.03.xsd` (sha256
// f8c8defb77c1c00928d0002cb2325b7aea2c9b2270a6ac8f572207ae2e11dfef, published
// 2020-02-13, ISO version 2013). Element order follows the XSD sequence.
type ForeignExchangeTradeWithdrawalNotificationV03 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.013.001.03 FXTradWdrwlNtfctn"`

	// Max35Text — message identification assigned by the sender.
	MessageID string `xml:"MsgId"`

	// Max35Text — matching system's unique reference for the trade.
	MatchingSysUniqueRef string `xml:"MtchgSysUnqRef"`

	// Optional reason for the withdrawal (minOccurs=0). Nil means no reason supplied.
	WithdrawalReason *WithdrawalReason1 `xml:"WdrwlRsn,omitempty"`

	// Optional settlement-session identifier (Exact4AlphaNumericText).
	SettlementSessionID string `xml:"SttlmSsnIdr,omitempty"`

	// Optional supplementary data payloads (minOccurs=0, maxOccurs=unbounded).
	SupplementaryData []SupplementaryData1 `xml:"SplmtryData,omitempty"`
}

// WithdrawalReason1 — reason code + optional sub-code.
//
// WdrwlRsnCd (required): one of RJCT (rejected), RSCD (rescinded), WTDN (withdrawn).
// WdrwlRsnSubCd (optional): Max4Text free-form sub-code.
type WithdrawalReason1 struct {
	Code    string `xml:"WdrwlRsnCd"`
	SubCode string `xml:"WdrwlRsnSubCd,omitempty"`
}

// SupplementaryData1 — ISO 20022 generic extension envelope.
//
// Carries an opaque payload (xs:any) that the CLS Business Guide reserves for
// bilateral extensions. The envelope is parsed as inner XML so round-trip is
// lossless even when the payload uses a different namespace.
type SupplementaryData1 struct {
	PlaceAndName string                     `xml:"PlcAndNm,omitempty"`
	Envelope     SupplementaryDataEnvelope1 `xml:"Envlp"`
}

// SupplementaryDataEnvelope1 — opaque xs:any content container.
type SupplementaryDataEnvelope1 struct {
	InnerXML string `xml:",innerxml"`
}
