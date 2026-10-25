// Package fxtr contains Go structs for the ISO 20022 fxtr (Foreign Exchange Trade) namespace.
//
// Coverage status (MS-023a):
//
//	fxtr.008.001.07  ✅ FXTradeNotificationV07         (CLS settlement queue ack)
//	fxtr.013.001.03  ✅ ForeignExchangeTradeWithdrawalNotificationV03 (CLS withdrawal)
//	fxtr.014.001.05  ✅ FXTradeCaptureConfirmationV05  (CLS — primary spot/forward)
//	fxtr.015.001.05  ✅ FXTradeAmendmentConfirmationV05 (CLS)
//	fxtr.016.001.05  ✅ FXTradeCancellationConfirmationV05 (CLS)
//	fxtr.017.001.05  ✅ FXTradeStatusReportV05         (CLS lifecycle PAIRED/SETTLED/etc.)
//	fxtr.030.001.05  ✅ FXSettlementNotificationV05    (CLS PvP terminal-state)
//	fxtr.031.001.02  ✅ FXTradeCaptureRequest031V02      (CFETS — member submits trade)
//	fxtr.032.001.02  ✅ FXTradeCaptureAck032V02          (CFETS → member; SUCCESS|REJECT)
//	fxtr.033.001.02  ✅ FXTradeCaptureNotification033V02 (CFETS → counterparty side)
//	fxtr.034.001.02  ✅ FXTradeConfirmationRequest034V02 (member → CFETS)
//	fxtr.035.001.02  ✅ FXTradeConfirmation035V02        (CFETS → both sides)
//	fxtr.036.001.02  ✅ FXTradeConfirmationStatus036V02  (CFETS → submitter; PAIRED|UNPAIRED|REJECT)
//	fxtr.037.001.02  ✅ FXTradeAmendment037V02           (member → CFETS amendment)
//	fxtr.038.001.02  ✅ FXTradeCancellation038V02        (member → CFETS cancellation)
//
// Design notes:
//
//   - These structs cover the COMMERCIAL fields actually exchanged with CLS in
//     spot/forward confirmation flow. Many optional XSD elements (e.g. complex
//     regulatory reporting blocks not exercised by CLS) are intentionally omitted.
//     They can be added per ticket as real-world traffic exercises them.
//   - All monetary amounts use `decimal.Decimal` (shopspring/decimal). XML
//     marshalling goes through helper types `Amount` and `Rate` which serialise
//     as plain decimal strings — NEVER float — preserving precision.
//   - Date/time fields use `ISODate` and `ISODateTime` helpers that round-trip
//     to the canonical ISO 20022 lexical forms (YYYY-MM-DD and YYYY-MM-DDTHH:mm:ss.fffZ).
//
// XSD pinning: see ../registry/sources.go. Actual XSDs are downloaded by
// scripts/download-xsd.sh into .cache/xsd/ and never embedded in the binary.
package fxtr
