//go:build grpcgen

package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/revenutech/exchangeos/modules/compliance/application"
	"github.com/revenutech/exchangeos/modules/compliance/domain"
	"github.com/revenutech/exchangeos/pkg/bacen"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

type GRPCServer struct{ svc *application.Service }

func NewGRPCServer(svc *application.Service) *GRPCServer { return &GRPCServer{svc: svc} }

// ClassifyOperation — pb.ComplianceServiceServer
//
// code_or_hint is required. It previously defaulted to the literal "10001",
// which silently classified every operation as "Exportação de mercadorias"
// regardless of what the caller sent — a misreport to BACEN, and one the caller
// had no way to detect. An absent hint is now an InvalidArgument: the server
// must never pick a nature code on the caller's behalf.
func (s *GRPCServer) ClassifyOperation(ctx context.Context, req *pb.ClassifyOperationRequest) (*pb.ClassifyOperationResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	trid, err := uuid.Parse(req.GetTradeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trade_id: %v", err)
	}
	hint := strings.TrimSpace(req.GetCodeOrHint())
	if hint == "" {
		return nil, status.Error(codes.InvalidArgument,
			"code_or_hint is required: the nature code cannot be inferred server-side")
	}

	c, err := s.svc.ClassifyOperation(ctx, tid, trid, hint)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.ClassifyOperationResponse{Classification: &pb.OperationClassification{
		ClassificationId: c.ID().String(),
		TradeId:          c.TradeID().String(),
		Code:             c.Code(),
		Description:      c.Description(),
		Nature:           string(c.Nature()),
	}}, nil
}

// ComputeIOF — pb.ComplianceServiceServer
func (s *GRPCServer) ComputeIOF(ctx context.Context, req *pb.ComputeIOFRequest) (*pb.ComputeIOFResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	trid, err := uuid.Parse(req.GetTradeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trade_id: %v", err)
	}

	// IOF is a tax. Every input previously carried a hardcoded default —
	// operation type "DEFAULT", notional 10000, currency "USD" — so the service
	// computed and PERSISTED a tax figure from invented numbers, whatever the
	// caller actually traded. All three are now required.
	opType := strings.TrimSpace(req.GetOperationType())
	if opType == "" {
		return nil, status.Error(codes.InvalidArgument, "operation_type is required")
	}
	money := req.GetNotional()
	if money == nil {
		return nil, status.Error(codes.InvalidArgument, "notional is required")
	}
	ccy := strings.TrimSpace(money.GetCurrency())
	if ccy == "" {
		return nil, status.Error(codes.InvalidArgument, "notional.currency is required")
	}
	notional, err := decimal.NewFromString(strings.TrimSpace(money.GetAmount()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "notional.amount: %v", err)
	}
	if !notional.IsPositive() {
		return nil, status.Error(codes.InvalidArgument, "notional.amount must be greater than zero")
	}

	iof, err := s.svc.ComputeIOF(ctx, tid, trid, opType, notional, ccy)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.ComputeIOFResponse{Iof: &pb.IOFComputation{
		IofId:         iof.ID().String(),
		TradeId:       iof.TradeID().String(),
		IofAmount:     &pb.Money{Amount: iof.IOFAmount().String(), Currency: iof.NotionalCCY()},
		RateApplied:   iof.Rate().String(),
		OperationType: iof.OperationType(),
	}}, nil
}

// SubmitBACENReport — pb.ComplianceServiceServer
func (s *GRPCServer) SubmitBACENReport(ctx context.Context, req *pb.SubmitBACENReportRequest) (*pb.SubmitBACENReportResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	rep := req.GetReport()
	if rep == nil {
		return nil, status.Error(codes.InvalidArgument, "report required")
	}
	// An empty payload used to be accepted and stored under the literal hash
	// "empty-payload". A BACEN report with no payload carries nothing to attest.
	if len(rep.GetPayload()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "report.payload is required")
	}
	r, err := s.svc.SubmitBACENReport(ctx, domain.NewBACENReportInput{
		TenantID:      tid,
		ReportType:    domain.ReportType(rep.GetReportType()),
		ReferenceDate: rep.GetReferenceDate().AsTime(),
		PayloadHash:   payloadHash(rep.GetPayload()),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.SubmitBACENReportResponse{Report: &pb.BACENReport{
		ReportId:   r.ID().String(),
		TenantId:   r.TenantID().String(),
		ReportType: string(r.Type()),
		Status:     string(r.Status()),
	}}, nil
}

// ScreenCounterparty — pb.ComplianceServiceServer
func (s *GRPCServer) ScreenCounterparty(ctx context.Context, req *pb.ScreenCounterpartyRequest) (*pb.ScreenCounterpartyResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	// Hits are resolved by the service's SanctionsScreener; the adapter must not
	// supply them. Without a provider wired the call fails closed.
	res, err := s.svc.ScreenCounterparty(ctx, domain.NewScreeningInput{
		TenantID: tid, CounterpartyBIC: req.GetCounterpartyBic(), LEI: req.GetLei(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.ScreenCounterpartyResponse{Result: &pb.ScreeningResult{
		Clear:     res.IsClear(),
		Hits:      res.Hits(),
		RiskLevel: string(res.RiskLevel()),
	}}, nil
}

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

// payloadHash returns the SHA-256 digest of the report payload, lowercase hex.
//
// It used to return string(payload[:32]) — the raw leading bytes, not a digest.
// That is the integrity anchor of a BACEN report: two different reports sharing
// a 32-byte prefix collided onto the same "hash", and the stored value echoed
// payload content instead of attesting to it.
func payloadHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, application.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, bacen.ErrUnknown):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// Force time import (used by upstream marshalling in proto adapters).
var _ = time.Now
