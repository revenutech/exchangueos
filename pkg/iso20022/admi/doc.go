// Package admi — ISO 20022 Administration (system event + reject + reference data) messages.
//
// Coverage (CLS):
//
//	admi.002.001.01  MessageReject                — generic reject (badly-formed or rule-violating msg)
//	admi.004.001.02  SystemEventNotification      — cycle-open, cycle-close, degraded, recovered…
//	admi.009.001.01  StaticDataRequest            — request for ref data refresh
//	admi.010.001.01  StaticDataReport             — response carrying ref data
//	admi.011.001.01  SystemEventAcknowledgement   — ack of system event
//	admi.017.001.02  AdministrationProprietary    — CLS-specific extension envelope
//	admi.024.001.01  NotificationOfCorrespondence — CLS member ops correspondence
package admi

import "encoding/xml"

// ── Common types ───────────────────────────────────────────────────────────

type ISODateTime struct {
	Value string `xml:",chardata"` // RFC3339; parsing/normalisation handled by callers
}

// PartyID — BICFI (and optional LEI) wrapper.
type PartyID struct {
	BICFI string `xml:"BICFI,omitempty"`
	LEI   string `xml:"LEI,omitempty"`
}

// ── admi.002.001.01 — MessageReject ────────────────────────────────────────

const Namespace002 = "urn:iso:std:iso:20022:tech:xsd:admi.002.001.01"

type MessageRejectV01 struct {
	XMLName       xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.002.001.01 MsgRjct"`
	RejectingPty  PartyID     `xml:"RjctgPty"`
	OriginalMsgID string      `xml:"OrgnlMsgId"`
	ReasonCode    string      `xml:"RsnCd"`
	ReasonText    string      `xml:"RsnTxt,omitempty"`
	RejectedAt    ISODateTime `xml:"RjctnDtTm"`
}

// ── admi.004.001.02 — SystemEventNotification ──────────────────────────────

const Namespace004 = "urn:iso:std:iso:20022:tech:xsd:admi.004.001.02"

type SystemEventNotificationV02 struct {
	XMLName    xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.004.001.02 SysEvtNtfctn"`
	EventCode  string      `xml:"EvtCd"` // e.g. "STRT","STOP","DEGR","RCVR","CYCO","CYCC"
	Component  string      `xml:"Cmpnt,omitempty"`
	OccurredAt ISODateTime `xml:"EvtDtTm"`
	Detail     string      `xml:"EvtDesc,omitempty"`
}

// ── admi.009.001.01 — StaticDataRequest ────────────────────────────────────

const Namespace009 = "urn:iso:std:iso:20022:tech:xsd:admi.009.001.01"

type StaticDataRequestV01 struct {
	XMLName   xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.009.001.01 StatcDataReq"`
	Requester PartyID     `xml:"Reqstr"`
	DataSet   string      `xml:"DataSet"` // e.g. "SSI", "CALENDAR", "BIC", "CURRENCY"
	AsOfDate  string      `xml:"AsOfDt,omitempty"`
	IssuedAt  ISODateTime `xml:"IssDtTm"`
}

// ── admi.010.001.01 — StaticDataReport ─────────────────────────────────────

const Namespace010 = "urn:iso:std:iso:20022:tech:xsd:admi.010.001.01"

type StaticDataReportV01 struct {
	XMLName     xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.010.001.01 StatcDataRpt"`
	ReportID    string      `xml:"RptId"`
	DataSet     string      `xml:"DataSet"`
	GeneratedAt ISODateTime `xml:"GnrtnDtTm"`
	PayloadXML  string      `xml:",innerxml"` // opaque body (XML or base64 BLOB)
}

// ── admi.011.001.01 — SystemEventAcknowledgement ───────────────────────────

const Namespace011 = "urn:iso:std:iso:20022:tech:xsd:admi.011.001.01"

type SystemEventAcknowledgementV01 struct {
	XMLName xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.011.001.01 SysEvtAck"`
	EventID string      `xml:"EvtId"`
	AckBy   PartyID     `xml:"AckBy"`
	AckAt   ISODateTime `xml:"AckDtTm"`
	Status  string      `xml:"Sts,omitempty"`
}

// ── admi.017.001.02 — AdministrationProprietary ────────────────────────────

const Namespace017Admi = "urn:iso:std:iso:20022:tech:xsd:admi.017.001.02"

type AdministrationProprietaryV02 struct {
	XMLName    xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:admi.017.001.02 AdmstnPrtry"`
	Originator PartyID  `xml:"Orgtr"`
	Subject    string   `xml:"Sbjct"`
	PayloadXML string   `xml:",innerxml"` // CLS-specific extension
}

// ── admi.024.001.01 — NotificationOfCorrespondence ─────────────────────────

const Namespace024 = "urn:iso:std:iso:20022:tech:xsd:admi.024.001.01"

type NotificationOfCorrespondenceV01 struct {
	XMLName  xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:admi.024.001.01 NtfctnOfCrspdc"`
	From     PartyID     `xml:"Fr"`
	To       PartyID     `xml:"To"`
	Subject  string      `xml:"Sbjct"`
	Body     string      `xml:"Body"`
	IssuedAt ISODateTime `xml:"IssDtTm"`
}
