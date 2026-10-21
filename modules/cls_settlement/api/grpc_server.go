//go:build grpcgen

package api

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	clsapp "github.com/revenutech/exchangeos/modules/cls_settlement/application"
	clsdomain "github.com/revenutech/exchangeos/modules/cls_settlement/domain"
	netapp "github.com/revenutech/exchangeos/modules/netreport/application"
	netdomain "github.com/revenutech/exchangeos/modules/netreport/domain"
	payapp "github.com/revenutech/exchangeos/modules/payin/application"
	paydomain "github.com/revenutech/exchangeos/modules/payin/domain"
	"github.com/revenutech/exchangeos/pkg/iso20022/camt"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// GRPCServer adapts pb.SettlementServiceServer over the cls_settlement + payin + netreport
// application services.
type GRPCServer struct {
	Cycles     *clsapp.Service
	PayIns     *payapp.Service
	NetReports *netapp.Service
}

func NewGRPCServer(cycles *clsapp.Service, payins *payapp.Service, nets *netapp.Service) *GRPCServer {
	return &GRPCServer{Cycles: cycles, PayIns: payins, NetReports: nets}
}

// OpenCycle — pb.SettlementServiceServer
func (s *GRPCServer) OpenCycle(ctx context.Context, req *pb.OpenCycleRequest) (*pb.OpenCycleResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	c, err := s.Cycles.OpenCycle(ctx, tid, req.GetCycleDate().AsTime())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.OpenCycleResponse{Cycle: toPBCycle(c)}, nil
}

// SubmitPayIn — pb.SettlementServiceServer
func (s *GRPCServer) SubmitPayIn(ctx context.Context, req *pb.SubmitPayInRequest) (*pb.SubmitPayInResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	in := req.GetInstruction()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "instruction required")
	}
	amt, err := decimal.NewFromString(in.GetAmount().GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "amount: %v", err)
	}
	cycleID, err := uuid.Parse(in.GetCycleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cycle_id: %v", err)
	}
	tenantID, err := uuid.Parse(in.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "tenant_id: %v", err)
	}

	created, err := s.PayIns.Create(ctx, paydomain.NewPayInInput{
		TenantID: tenantID,
		CycleID:  cycleID,
		Currency: in.GetCurrency(),
		Amount:   amt,
		Band:     paydomain.BandPIN3, // default; proto does not yet carry band — caller may extend
		Deadline: in.GetDeadline().AsTime(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	submittedAt := in.GetSubmittedAt().AsTime()
	if submittedAt.IsZero() {
		submittedAt = time.Now().UTC()
	}
	submitted, err := s.PayIns.Submit(ctx, created.ID(), submittedAt)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.SubmitPayInResponse{Instruction: toPBPayIn(submitted)}, nil
}

// GetNetReport — pb.SettlementServiceServer. Marshals the persisted net lines
// into a camt.088.001.02 NetReport.
func (s *GRPCServer) GetNetReport(ctx context.Context, req *pb.GetNetReportRequest) (*pb.GetNetReportResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	cycleID, err := uuid.Parse(req.GetCycleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cycle_id: %v", err)
	}
	memberBIC := strings.ToUpper(strings.TrimSpace(req.GetMemberBic()))
	if l := len(memberBIC); l != 8 && l != 11 {
		return nil, status.Error(codes.InvalidArgument, "member_bic must be 8 or 11 characters")
	}

	list, err := s.NetReports.ListByCycle(ctx, cycleID)
	if err != nil {
		return nil, mapErr(err)
	}
	xmlBody, err := renderNetReportLines(list, memberBIC)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}
	return &pb.GetNetReportResponse{
		ReportXml:   xmlBody,
		GeneratedAt: timestamppb.Now(),
	}, nil
}

// CloseCycle — pb.SettlementServiceServer (advances PAY_IN_WINDOW → SETTLING → CLOSED).
func (s *GRPCServer) CloseCycle(ctx context.Context, req *pb.CloseCycleRequest) (*pb.CloseCycleResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	cycleID, err := uuid.Parse(req.GetCycleId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cycle_id: %v", err)
	}
	now := time.Now().UTC()
	if _, err := s.Cycles.EnterSettling(ctx, cycleID, now); err != nil {
		return nil, mapErr(err)
	}
	c, err := s.Cycles.CloseCycle(ctx, cycleID, now)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CloseCycleResponse{Cycle: toPBCycle(c)}, nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func parseTenant(t *pb.TenantContext) (uuid.UUID, error) {
	if t == nil {
		return uuid.Nil, status.Error(codes.InvalidArgument, "tenant context required")
	}
	tid, err := uuid.Parse(t.GetTenantId())
	if err != nil {
		return uuid.Nil, status.Errorf(codes.InvalidArgument, "tenant_id: %v", err)
	}
	return tid, nil
}

func toPBCycle(c *clsdomain.CLSCycle) *pb.CLSCycle {
	out := &pb.CLSCycle{
		CycleId:  c.ID().String(),
		TenantId: c.TenantID().String(),
		Status:   pb.CycleStatus(pb.CycleStatus_value["CYCLE_STATUS_"+string(c.Status())]),
		OpenedAt: timestamppb.New(c.OpenedAt()),
		// proto.closed_at is the scheduled-close per the message comment in settlement.proto.
		ClosedAt: timestamppb.New(c.ScheduledClose()),
	}
	if d, err := c.DeadlineFor("PIN1"); err == nil {
		out.PayInDeadline_1 = timestamppb.New(d)
	}
	if d, err := c.DeadlineFor("PIN2"); err == nil {
		out.PayInDeadline_2 = timestamppb.New(d)
	}
	if d, err := c.DeadlineFor("PIN3"); err == nil {
		out.PayInDeadline_3 = timestamppb.New(d)
	}
	for _, id := range c.TradeIDs() {
		out.TradeIds = append(out.TradeIds, id.String())
	}
	return out
}

func toPBPayIn(p *paydomain.PayInInstruction) *pb.PayInInstruction {
	out := &pb.PayInInstruction{
		InstructionId: p.ID().String(),
		CycleId:       p.CycleID().String(),
		TenantId:      p.TenantID().String(),
		Currency:      p.Currency(),
		Amount:        &pb.Money{Amount: p.Amount().String(), Currency: p.Currency()},
		Deadline:      timestamppb.New(p.Deadline()),
		Status:        string(p.Status()),
	}
	if !p.SubmittedAt().IsZero() {
		out.SubmittedAt = timestamppb.New(p.SubmittedAt())
	}
	return out
}

// renderNetReportLines marshals the persisted net lines into a camt.088.001.02
// NetReport.
//
// It used to return the literal string "<NetRpt placeholder/>" — a settlement
// report with no cycle, no member and no amounts, handed to the caller as if it
// were the real end-of-cycle net position.
//
// An empty slice yields an error rather than an empty report: "no lines" and
// "nothing owed" are different claims, and only the caller knows which cycle it
// asked about.
func renderNetReportLines(reports []*netdomain.NetReport, memberBIC string) (string, error) {
	if len(reports) == 0 {
		return "", fmt.Errorf("no net report lines for the requested cycle")
	}

	first := reports[0]
	rpt := camt.NetReportV02{
		ReportID: first.ID().String(),
		CycleID:  first.CycleID().String(),
		Member:   camt.PartyID{BICFI: memberBIC},
		GeneratedAt: camt.ISODateTime{
			Value: first.GeneratedAt().UTC().Format(time.RFC3339),
		},
		Lines: make([]camt.NetLine, 0, len(reports)),
	}

	for _, r := range reports {
		if r.CycleID() != first.CycleID() {
			return "", fmt.Errorf("net report lines span multiple cycles (%s and %s)",
				first.CycleID(), r.CycleID())
		}
		ccy := r.Currency()
		rpt.Lines = append(rpt.Lines, camt.NetLine{
			Currency:      ccy,
			GrossPayIn:    camt.Amount{Currency: ccy, Value: r.GrossPayIn()},
			GrossPayOut:   camt.Amount{Currency: ccy, Value: r.GrossPayOut()},
			NetSettlement: camt.Amount{Currency: ccy, Value: r.NetSettlement()},
			TradeCount:    r.TradeCount(),
		})
	}

	out, err := xml.Marshal(rpt)
	if err != nil {
		return "", fmt.Errorf("marshal camt.088 net report: %w", err)
	}
	return xml.Header + string(out), nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, clsapp.ErrNotFound), errors.Is(err, payapp.ErrNotFound), errors.Is(err, netapp.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, clsapp.ErrInvalidInput), errors.Is(err, payapp.ErrInvalidInput), errors.Is(err, netapp.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, clsapp.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, clsdomain.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, paydomain.ErrDeadlineMissed):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
