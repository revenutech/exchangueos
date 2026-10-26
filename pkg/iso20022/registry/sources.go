package registry

// Default catalog wiring for ExchangeOS — 32 FX-specific schemas pinned by version.
// XSDs are NOT bundled in the binary; URLs feed scripts/download-xsd.sh + CI verification.
//
// References:
//   - ISO 20022 official: https://www.iso20022.org/iso-20022-message-definitions
//   - CLS: https://www.cls-group.com/products/cls-data/iso-20022/
//   - CFETS: https://www.chinamoney.com.cn/english/
//
// Versions chosen are the latest stable as of 2026-05-24 — bump deliberately
// when CLS / CFETS publish new variants.
//
// URLS ARE EXPLICIT LITERALS (no string concatenation). The download-xsd.sh
// script greps this file for `XSDSourceURL:` lines — if you add concatenation,
// you break the catalog tooling.

// Default returns a Registry pre-populated with all 32 ExchangeOS schemas.
func Default() *Registry {
	r := New()

	// ── fxtr CLS (7) ────────────────────────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "008", Variant: "001", Version: "07",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.008.001.07.xsd",
		Description:  "Foreign Exchange Trade Notification (CLS settlement queue)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "013", Variant: "001", Version: "03",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.013.001.03.xsd",
		Description:  "Trade Withdrawal Notification (CLS)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "014", Variant: "001", Version: "05",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.014.001.05.xsd",
		Description:  "FX Trade Capture Confirmation (CLS) — primary spot/forward confirmation"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "015", Variant: "001", Version: "05",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.015.001.05.xsd",
		Description:  "FX Trade Amendment Confirmation (CLS)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "016", Variant: "001", Version: "05",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.016.001.05.xsd",
		Description:  "FX Trade Cancellation Confirmation (CLS)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "017", Variant: "001", Version: "05",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.017.001.05.xsd",
		Description:  "FX Trade Status Report (CLS)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessFXTR, MessageDef: "030", Variant: "001", Version: "05",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/fxtr/schemas/fxtr.030.001.05.xsd",
		Description:  "CLS Settlement Notification"})

	// ── fxtr CFETS (8) ──────────────────────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "031", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.031.001.02.xsd",
		Description:  "CFETS Trade Capture Request (PTPP)"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "032", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.032.001.02.xsd",
		Description:  "CFETS Trade Capture Acknowledgement"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "033", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.033.001.02.xsd",
		Description:  "CFETS Trade Capture Notification"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "034", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.034.001.02.xsd",
		Description:  "CFETS Trade Confirmation Request"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "035", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.035.001.02.xsd",
		Description:  "CFETS Trade Confirmation"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "036", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.036.001.02.xsd",
		Description:  "CFETS Trade Confirmation Status"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "037", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.037.001.02.xsd",
		Description:  "CFETS Trade Amendment"})
	r.MustRegister(Descriptor{Organisation: OrgCFETS, Domain: BusinessFXTR, MessageDef: "038", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.chinamoney.com.cn/iso/fxtr.038.001.02.xsd",
		Description:  "CFETS Trade Cancellation"})

	// ── admi CLS (6) ────────────────────────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "002", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.002.001.01.xsd",
		Description:  "admi.002 — Message Reject"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "004", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.004.001.02.xsd",
		Description:  "admi.004 — System Event Notification"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "009", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.009.001.01.xsd",
		Description:  "admi.009 — Static Data Request"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "010", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.010.001.01.xsd",
		Description:  "admi.010 — Static Data Report"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "011", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.011.001.01.xsd",
		Description:  "admi.011 — System Event Acknowledgement"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "017", Variant: "001", Version: "02",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.017.001.02.xsd",
		Description:  "admi.017 — Administration Proprietary Message"})

	// ── camt CLS PayIn + NetReport (4) ──────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessCAMT, MessageDef: "061", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/camt/schemas/camt.061.001.01.xsd",
		Description:  "camt.061 — PayIn Schedule (CLS)"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessCAMT, MessageDef: "062", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/camt/schemas/camt.062.001.01.xsd",
		Description:  "camt.062 — PayIn Schedule Acknowledgement"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessCAMT, MessageDef: "063", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/camt/schemas/camt.063.001.01.xsd",
		Description:  "camt.063 — PayIn Schedule Cancellation"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessCAMT, MessageDef: "088", Variant: "001", Version: "03",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/camt/schemas/camt.088.001.03.xsd",
		Description:  "camt.088 — Net Report (CLS settlement summary)"})

	// ── reda CLS (2) ────────────────────────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessREDA, MessageDef: "060", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/reda/schemas/reda.060.001.01.xsd",
		Description:  "reda.060 — Settlement Standing Instruction Request"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessREDA, MessageDef: "061", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/reda/schemas/reda.061.001.01.xsd",
		Description:  "reda.061 — Settlement Standing Instruction Confirmation"})

	// ── head + supporting (5) ───────────────────────────────────────────
	r.MustRegister(Descriptor{Organisation: OrgISO, Domain: BusinessHEAD, MessageDef: "001", Variant: "001", Version: "03",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/head/schemas/head.001.001.03.xsd",
		Description:  "Business Application Header (BAH)"})
	r.MustRegister(Descriptor{Organisation: OrgISO, Domain: BusinessHEAD, MessageDef: "002", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/head/schemas/head.002.001.01.xsd",
		Description:  "BAH Acknowledgement"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessREDA, MessageDef: "066", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/reda/schemas/reda.066.001.01.xsd",
		Description:  "Calendar/Holiday Reference Data"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessREDA, MessageDef: "067", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/reda/schemas/reda.067.001.01.xsd",
		Description:  "SSI Reference Data Update"})
	r.MustRegister(Descriptor{Organisation: OrgCLS, Domain: BusinessADMI, MessageDef: "024", Variant: "001", Version: "01",
		XSDSourceURL: "https://www.iso20022.org/sites/default/files/documents/messages/admi/schemas/admi.024.001.01.xsd",
		Description:  "admi.024 — Notification of Correspondence (CLS member ops)"})

	return r
}
