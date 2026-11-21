package fxtr

import "encoding/xml"

// CFETS PTPP (Post-Trade Processing Platform) fxtr family — variant `.001.02`.
// References: https://www.chinamoney.com.cn/english/  (CFETS publications).
//
// Coverage:
//
//	fxtr.031 Trade Capture Request          (member → CFETS)
//	fxtr.032 Trade Capture Acknowledgement  (CFETS → member; SUCCESS|REJECT)
//	fxtr.033 Trade Capture Notification     (CFETS → counterparty side)
//	fxtr.034 Trade Confirmation Request     (member → CFETS)
//	fxtr.035 Trade Confirmation             (CFETS → both sides)
//	fxtr.036 Trade Confirmation Status      (CFETS → submitter; PAIRED|UNPAIRED|REJECT)
//	fxtr.037 Trade Amendment                (member → CFETS)
//	fxtr.038 Trade Cancellation             (member → CFETS)

// ─────────────────────────────────────────────────────────────────────────────
// Shared CFETS payload — most messages echo the same trade identification +
// economics block. Variant fields carry status / reason / amendment fields.
// ─────────────────────────────────────────────────────────────────────────────

// CFETSTradeIdentification — CFETS uses its own deal id alongside the submitter's ref.
type CFETSTradeIdentification struct {
	CFETSDealID  string      `xml:"CfetDealId"` // 16-char CFETS deal id
	SubmitterRef string      `xml:"SubmRef,omitempty"`
	CapturedAt   ISODateTime `xml:"CaptrDtTm,omitempty"`
}

// CFETSEconomics — common trade economics block.
type CFETSEconomics struct {
	TradeDate  ISODate       `xml:"TradDt"`
	ValueDate  ISODate       `xml:"ValDt"`
	TradedAmts TradedAmounts `xml:"TraddAmts"`
	AgreedRate AgreedRate    `xml:"AgrdRate"`
}

// ─── fxtr.031.001.02 — Trade Capture Request ───────────────────────────────

const Namespace031 = "urn:iso:std:iso:20022:tech:xsd:fxtr.031.001.02"

type FXTradeCaptureRequest031V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.031.001.02 FXTradCaptrReq"`

	TradeID  CFETSTradeIdentification `xml:"TradId"`
	Trading  SideIdentification       `xml:"TradgSdIdr"`
	Counterp SideIdentification       `xml:"CtrPtySdIdr"`
	Econ     CFETSEconomics           `xml:"Econs"`
}

// ─── fxtr.032.001.02 — Trade Capture Acknowledgement ───────────────────────

const Namespace032 = "urn:iso:std:iso:20022:tech:xsd:fxtr.032.001.02"

type CFETSAckStatus string

const (
	CFETSAckSuccess CFETSAckStatus = "SUCC"
	CFETSAckReject  CFETSAckStatus = "REJT"
)

type FXTradeCaptureAck032V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.032.001.02 FXTradCaptrAck"`

	TradeID      CFETSTradeIdentification `xml:"TradId"`
	Status       CFETSAckStatus           `xml:"Sts"`
	StatusReason string                   `xml:"StsRsn,omitempty"`
	AckAt        ISODateTime              `xml:"AckDtTm"`
}

// ─── fxtr.033.001.02 — Trade Capture Notification ──────────────────────────

const Namespace033 = "urn:iso:std:iso:20022:tech:xsd:fxtr.033.001.02"

type FXTradeCaptureNotification033V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.033.001.02 FXTradCaptrNtfctn"`

	TradeID  CFETSTradeIdentification `xml:"TradId"`
	Trading  SideIdentification       `xml:"TradgSdIdr"`
	Counterp SideIdentification       `xml:"CtrPtySdIdr"`
	Econ     CFETSEconomics           `xml:"Econs"`
	NotifyAt ISODateTime              `xml:"NtfDtTm"`
}

// ─── fxtr.034.001.02 — Trade Confirmation Request ──────────────────────────

const Namespace034 = "urn:iso:std:iso:20022:tech:xsd:fxtr.034.001.02"

type FXTradeConfirmationRequest034V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.034.001.02 FXTradConfReq"`

	TradeID  CFETSTradeIdentification `xml:"TradId"`
	Trading  SideIdentification       `xml:"TradgSdIdr"`
	Counterp SideIdentification       `xml:"CtrPtySdIdr"`
	Econ     CFETSEconomics           `xml:"Econs"`
}

// ─── fxtr.035.001.02 — Trade Confirmation ──────────────────────────────────

const Namespace035 = "urn:iso:std:iso:20022:tech:xsd:fxtr.035.001.02"

type FXTradeConfirmation035V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.035.001.02 FXTradConf"`

	TradeID     CFETSTradeIdentification `xml:"TradId"`
	Trading     SideIdentification       `xml:"TradgSdIdr"`
	Counterp    SideIdentification       `xml:"CtrPtySdIdr"`
	Econ        CFETSEconomics           `xml:"Econs"`
	ConfirmedAt ISODateTime              `xml:"ConfDtTm"`
}

// ─── fxtr.036.001.02 — Trade Confirmation Status ───────────────────────────

const Namespace036 = "urn:iso:std:iso:20022:tech:xsd:fxtr.036.001.02"

type CFETSConfStatus string

const (
	CFETSConfPaired   CFETSConfStatus = "PAIR"
	CFETSConfUnpaired CFETSConfStatus = "UPRD"
	CFETSConfReject   CFETSConfStatus = "REJT"
)

type FXTradeConfirmationStatus036V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.036.001.02 FXTradConfSts"`

	TradeID      CFETSTradeIdentification `xml:"TradId"`
	Status       CFETSConfStatus          `xml:"Sts"`
	StatusReason string                   `xml:"StsRsn,omitempty"`
	StatusAt     ISODateTime              `xml:"StsDtTm"`
}

// ─── fxtr.037.001.02 — Trade Amendment ─────────────────────────────────────

const Namespace037 = "urn:iso:std:iso:20022:tech:xsd:fxtr.037.001.02"

type FXTradeAmendment037V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.037.001.02 FXTradAmdmnt"`

	OriginalTradeID  CFETSTradeIdentification `xml:"OrgnlTradId"`
	Trading          SideIdentification       `xml:"TradgSdIdr"`
	Counterp         SideIdentification       `xml:"CtrPtySdIdr"`
	AmendedEconomics CFETSEconomics           `xml:"AmddEcons"`
	AmendmentReason  string                   `xml:"AmdmntRsn,omitempty"`
	AmendedAt        ISODateTime              `xml:"AmdmntDtTm"`
}

// ─── fxtr.038.001.02 — Trade Cancellation ──────────────────────────────────

const Namespace038 = "urn:iso:std:iso:20022:tech:xsd:fxtr.038.001.02"

type FXTradeCancellation038V02 struct {
	XMLName xml.Name `xml:"urn:iso:std:iso:20022:tech:xsd:fxtr.038.001.02 FXTradCxl"`

	OriginalTradeID    CFETSTradeIdentification `xml:"OrgnlTradId"`
	Trading            SideIdentification       `xml:"TradgSdIdr"`
	Counterp           SideIdentification       `xml:"CtrPtySdIdr"`
	CancellationReason string                   `xml:"CxlRsn,omitempty"`
	CancelledAt        ISODateTime              `xml:"CxlDtTm"`
}
