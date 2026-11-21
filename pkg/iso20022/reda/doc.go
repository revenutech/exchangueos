// Package reda — ISO 20022 Reference Data messages (CLS settlement instructions + calendars).
//
// Coverage:
//
//	reda.060.001.01  SettlementStandingInstructionRequestV01
//	reda.061.001.01  SettlementStandingInstructionConfirmationV01
//	reda.066.001.01  CalendarReferenceDataV01     (holiday set per calendar_id)
//	reda.067.001.01  SSIReferenceDataUpdateV01    (bulk SSI refresh)
package reda

import "encoding/xml"

type ISODate struct {
	Value string `xml:",chardata"` // yyyy-MM-dd
}

type ISODateTime struct {
	Value string `xml:",chardata"` // RFC3339
}

type PartyID struct {
	BICFI string `xml:"BICFI,omitempty"`
	LEI   string `xml:"LEI,omitempty"`
	Name  string `xml:"Nm,omitempty"`
}

// ── reda.060.001.01 — SSI Request ──────────────────────────────────────────

const Namespace060 = "urn:iso:std:iso:20022:tech:xsd:reda.060.001.01"

type SettlementStandingInstructionRequestV01 struct {
	XMLName       xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:reda.060.001.01 SttlmStgInstrReq"`
	Requester     PartyID     `xml:"Reqstr"`
	Counterparty  PartyID     `xml:"CtrPty"`
	Currency      string      `xml:"Ccy"`
	EffectiveDate ISODate     `xml:"EfctvDt,omitempty"`
	IssuedAt      ISODateTime `xml:"IssDtTm"`
}

// ── reda.061.001.01 — SSI Confirmation ─────────────────────────────────────

const Namespace061 = "urn:iso:std:iso:20022:tech:xsd:reda.061.001.01"

type SettlementStandingInstructionConfirmationV01 struct {
	XMLName         xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:reda.061.001.01 SttlmStgInstrConf"`
	SSIID           string      `xml:"SSIId"`
	Requester       PartyID     `xml:"Reqstr"`
	Counterparty    PartyID     `xml:"CtrPty"`
	Currency        string      `xml:"Ccy"`
	BeneficiaryBIC  string      `xml:"BnfryBICFI"`
	IntermediaryBIC string      `xml:"IntrmyBICFI,omitempty"`
	AccountNumber   string      `xml:"AcctNb"`
	IBAN            string      `xml:"IBAN,omitempty"`
	ValidFrom       ISODate     `xml:"VldFr"`
	ValidTo         ISODate     `xml:"VldTo,omitempty"`
	ConfirmedAt     ISODateTime `xml:"ConfDtTm"`
}

// ── reda.066.001.01 — Calendar Reference Data ──────────────────────────────

const Namespace066 = "urn:iso:std:iso:20022:tech:xsd:reda.066.001.01"

type CalendarReferenceDataV01 struct {
	XMLName    xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:reda.066.001.01 CalRefData"`
	CalendarID string      `xml:"CalId"` // e.g. "USD_NYC", "EUR_TARGET2"
	Holidays   []ISODate   `xml:"Hlday"`
	IssuedAt   ISODateTime `xml:"IssDtTm"`
}

// ── reda.067.001.01 — SSI Reference Data Update (bulk) ─────────────────────

const Namespace067 = "urn:iso:std:iso:20022:tech:xsd:reda.067.001.01"

type SSIReferenceDataUpdateV01 struct {
	XMLName  xml.Name    `xml:"urn:iso:std:iso:20022:tech:xsd:reda.067.001.01 SSIRefDataUpd"`
	UpdateID string      `xml:"UpdId"`
	Updates  []SSIChange `xml:"Updt"`
	IssuedAt ISODateTime `xml:"IssDtTm"`
}

type SSIChange struct {
	Action   string `xml:"Actn"` // ADD | MOD | DEL
	SSIID    string `xml:"SSIId"`
	Currency string `xml:"Ccy"`
	BICFI    string `xml:"BICFI,omitempty"`
}
