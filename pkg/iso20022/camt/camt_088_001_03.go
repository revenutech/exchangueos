package camt

import "encoding/xml"

// Namespace088V03 — pinned schema namespace for camt.088.001.03 (official).
//
// camt.088 is the CLS NetReport — end-of-cycle summary of settlement
// obligations between a netting participant and CLS Bank. ISO 20022 SR2026
// lists .001.03 as the latest variant; the .001.04 referenced by earlier
// drafts of this package has no XSD in the public catalogue and is tracked
// as a plan gap.
//
// Fields mirror the XSD at `.cache/xsd/camt.088.001.03.xsd` (sha256
// 495c686343e29c58b8a9e47dbefc18091be9572981053b5f6f4001b281d28072,
// published 2025-03-01, ISO version 2013). The non-conformant NetReportV02
// in doc.go is kept, unreachable, as a git anchor for the fix.
const Namespace088V03 = "urn:iso:std:iso:20022:tech:xsd:camt.088.001.03"

// NetReportV03 — camt.088.001.03 root (inside the ISO 20022 <Document>).
//
// Mandatory elements (XSD sequence):
//
//	NetRptData       NetReportData2
//	NetSvcPtcptId    PartyIdentification242Choice
//	NetSvcCtrPtyId?  PartyIdentification242Choice   (0..1)
//	NetOblgtn        NetObligation3                 (1..∞)
//	SplmtryData?     SupplementaryData1             (0..∞)
type NetReportV03 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:camt.088.001.03 NetRpt"`

	ReportData             NetReportData2                `xml:"NetRptData"`
	NetServiceParticipant  PartyIdentification242Choice  `xml:"NetSvcPtcptId"`
	NetServiceCounterparty *PartyIdentification242Choice `xml:"NetSvcCtrPtyId,omitempty"`
	Obligations            []NetObligation3              `xml:"NetOblgtn"`
	SupplementaryData      []SupplementaryData1          `xml:"SplmtryData,omitempty"`
}

// NetReportData2 — report-level metadata.
//
// The three optional elements (NetRptSvcr, NetSvcTp, MsgPgntn) are omitted
// from the Go projection until an actual emitter needs them; they can be
// added later without breaking existing callers because Go's xml package
// ignores unknown fields on unmarshal.
type NetReportData2 struct {
	MessageID         string      `xml:"MsgId"`
	CreatedAt         ISODateTime `xml:"CreDtTm"`
	NettingCutOffTime ISOTime     `xml:"NetgCutOffTm"`
	ReportDate        ISODate     `xml:"RptDt"`
	ValueDate         ISODate     `xml:"ValDt"`
	ReportType        string      `xml:"RptTp,omitempty"`
}

// NetObligation3 — one netting obligation between participant and counterparty.
//
// Required: ObligationID, Amount, ParticipantNettingID, ObligationDirection,
// CounterpartyNettingID. Four optional fields (NetSvcCtrPtyId,
// CtrPtySttlmInstrs, PmtClrCentr, TxsNb) are omitted — the exchangeos domain
// does not carry per-obligation counterparty variation or clearing-centre
// routing yet.
type NetObligation3 struct {
	ObligationID          string                       `xml:"OblgtnId"`
	Amount                Amount                       `xml:"Amt"` // ActiveCurrencyAndAmount
	ParticipantNettingID  NettingIdentification2Choice `xml:"PtcptNetgId"`
	ObligationDirection   string                       `xml:"OblgtnDrctn"` // PAYM | RECE | NONE (PaymentReceipt1Code)
	CounterpartyNettingID NettingIdentification2Choice `xml:"CtrPtyNetgId"`
}

// PartyIdentification242Choice — one of three party-identification variants.
//
// The XSD defines this as an xs:choice between NmAndAdr, AnyBIC and PtyId.
// Emitters set exactly one of the three pointers; unmarshal populates the
// one that appeared in the XML.
type PartyIdentification242Choice struct {
	NameAndAddress *NameAndAddress8        `xml:"NmAndAdr,omitempty"`
	AnyBIC         *PartyIdentification265 `xml:"AnyBIC,omitempty"`
	PartyID        *PartyIdentification266 `xml:"PtyId,omitempty"`
}

// PartyIdentification265 — AnyBIC (ISO 20022 BIC) + optional alternative IDs.
//
// AlternativeIDs is kept as a slice for round-trip fidelity even though the
// exchangeos emitter never populates it today.
type PartyIdentification265 struct {
	AnyBIC         string   `xml:"AnyBIC"`
	AlternativeIDs []string `xml:"AltrntvIdr,omitempty"`
}

// NameAndAddress8 — minimal projection (Name only).
//
// The XSD also allows Adr (PostalAddress1) and AltrntvIdr; emitters that need
// them can wrap this type. The exchangeos emitter uses AnyBIC exclusively,
// so NameAndAddress8 participates only in the choice shape for unmarshal
// round-trips from external senders.
type NameAndAddress8 struct {
	Name string `xml:"Nm"`
}

// PartyIdentification266 — minimal projection (PtyNm only).
//
// Six XSD optional fields (PtyNm, AnyBIC, AcctNb, Adr, ClrSysId, LglNttyIdr)
// are summarised to PartyName here. Extended by future callers if needed.
type PartyIdentification266 struct {
	PartyName string `xml:"PtyNm,omitempty"`
}

// NettingIdentification2Choice — xs:choice between TradPty and NetgGrpId.
//
// The exchangeos emitter uses TradPty with an AnyBIC party. NetgGrpId is kept
// for round-trip fidelity.
type NettingIdentification2Choice struct {
	TradingParty   *PartyIdentification242Choice `xml:"TradPty,omitempty"`
	NettingGroupID string                        `xml:"NetgGrpId,omitempty"`
}

// SupplementaryData1 — ISO 20022 generic extension envelope.
type SupplementaryData1 struct {
	PlaceAndName string                     `xml:"PlcAndNm,omitempty"`
	Envelope     SupplementaryDataEnvelope1 `xml:"Envlp"`
}

// SupplementaryDataEnvelope1 — opaque xs:any payload.
type SupplementaryDataEnvelope1 struct {
	InnerXML string `xml:",innerxml"`
}

// ISODate — ISO 20022 xs:date (yyyy-MM-dd).
type ISODate struct {
	Value string `xml:",chardata"`
}

// ISOTime — ISO 20022 xs:time (HH:mm:ss[.sss]).
type ISOTime struct {
	Value string `xml:",chardata"`
}
