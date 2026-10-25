package api

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	clsdomain "github.com/revenutech/exchangeos/modules/cls_settlement/domain"
	netdomain "github.com/revenutech/exchangeos/modules/netreport/domain"
	"github.com/revenutech/exchangeos/pkg/iso20022/camt"
)

// clsBankBIC — CLS Bank N.A. ISO 20022 AnyBIC. CLS is the only counterparty on
// a netting participant's camt.088 obligations.
const clsBankBIC = "CLSBUS33"

// renderNetReportV03 builds the camt.088.001.03 body for one cycle.
//
// The emitter joins two aggregates by design (MS-079 7.5 part 2 design B):
//
//   - *clsdomain.CLSCycle    → carries NetgCutOffTm (PIN3 deadline, HH:mm:ss CET)
//     and the business date reused as RptDt + ValDt (CLS netting settles same-day).
//   - []*netdomain.NetReport → one aggregate per currency, each contributing one
//     <NetOblgtn>.
//
// Keeping the three missing ISO fields on the cycle instead of on NetReport
// avoids forking the domain to serve a report shape; the netreport aggregate
// stays single-currency and single-cycle.
func renderNetReportV03(cycle *clsdomain.CLSCycle, reports []*netdomain.NetReport, memberBIC string) (string, error) {
	if cycle == nil {
		return "", fmt.Errorf("cycle required")
	}
	if len(reports) == 0 {
		return "", fmt.Errorf("no net report lines for cycle %s", cycle.ID())
	}

	pin3, err := cycle.DeadlineFor("PIN3")
	if err != nil {
		return "", fmt.Errorf("cycle %s has no PIN3 deadline: %w", cycle.ID(), err)
	}
	cycleDate := cycle.CycleDate().UTC().Format("2006-01-02")

	member := strings.ToUpper(strings.TrimSpace(memberBIC))

	rpt := camt.NetReportV03{
		ReportData: camt.NetReportData2{
			MessageID: fmt.Sprintf("NR-%s-%s-%s",
				cycleDate, member, cycle.ID().String()[:8]),
			CreatedAt:         camt.ISODateTime{Value: reports[0].GeneratedAt().UTC().Format(time.RFC3339)},
			NettingCutOffTime: camt.ISOTime{Value: pin3.UTC().Format("15:04:05")},
			ReportDate:        camt.ISODate{Value: cycleDate},
			ValueDate:         camt.ISODate{Value: cycleDate},
		},
		NetServiceParticipant: camt.PartyIdentification242Choice{
			AnyBIC: &camt.PartyIdentification265{AnyBIC: member},
		},
		NetServiceCounterparty: &camt.PartyIdentification242Choice{
			AnyBIC: &camt.PartyIdentification265{AnyBIC: clsBankBIC},
		},
		Obligations: make([]camt.NetObligation3, 0, len(reports)),
	}

	for _, r := range reports {
		if r.CycleID() != cycle.ID() {
			return "", fmt.Errorf("net report %s belongs to cycle %s, not %s",
				r.ID(), r.CycleID(), cycle.ID())
		}
		direction := "NONE"
		switch {
		case r.IsReceivable():
			direction = "RECE"
		case r.IsPayable():
			direction = "PAYM"
		}
		rpt.Obligations = append(rpt.Obligations, camt.NetObligation3{
			ObligationID: r.ID().String(),
			Amount: camt.Amount{
				Currency: r.Currency(),
				Value:    r.NetSettlement().Abs(),
			},
			ParticipantNettingID: camt.NettingIdentification2Choice{
				TradingParty: &camt.PartyIdentification242Choice{
					AnyBIC: &camt.PartyIdentification265{AnyBIC: member},
				},
			},
			ObligationDirection: direction,
			CounterpartyNettingID: camt.NettingIdentification2Choice{
				TradingParty: &camt.PartyIdentification242Choice{
					AnyBIC: &camt.PartyIdentification265{AnyBIC: clsBankBIC},
				},
			},
		})
	}

	out, err := xml.Marshal(rpt)
	if err != nil {
		return "", fmt.Errorf("marshal camt.088.001.03 net report: %w", err)
	}
	return xml.Header + string(out), nil
}
