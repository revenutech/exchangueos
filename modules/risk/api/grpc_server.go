//go:build grpcgen

package api

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/revenutech/exchangeos/modules/risk/application"
	"github.com/revenutech/exchangeos/modules/risk/domain"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

type GRPCServer struct {
	svc *application.Service
}

func NewGRPCServer(svc *application.Service) *GRPCServer { return &GRPCServer{svc: svc} }

// CheckLimit — pb.RiskServiceServer
func (s *GRPCServer) CheckLimit(ctx context.Context, req *pb.CheckLimitRequest) (*pb.CheckLimitResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	if req.GetProposedExposure() == nil {
		return nil, status.Error(codes.InvalidArgument, "proposed_exposure required")
	}
	exposure, err := decimal.NewFromString(req.GetProposedExposure().GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "proposed_exposure.amount: %v", err)
	}
	// limit_type and scope used to be hardcoded to LimitCounterparty and the
	// trade_id. A trade id is not a limit scope — limits are configured per
	// counterparty BIC, currency or tenor — so repo.Find never matched and the
	// check returned NotFound for every call. RN_FX_015 was not enforced through
	// this RPC at all.
	limitType, err := parseLimitType(req.GetLimitType())
	if err != nil {
		return nil, err
	}
	scope := strings.TrimSpace(req.GetScope())
	if scope == "" {
		return nil, status.Error(codes.InvalidArgument,
			"scope is required (counterparty BIC, currency or tenor the limit applies to)")
	}

	res, err := s.svc.CheckLimit(ctx, tid, limitType, scope, exposure)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CheckLimitResponse{
		Allowed:        res.Allowed,
		BreachedLimits: res.BreachedLimits,
		Explanation:    res.Explanation,
	}, nil
}

// GetExposure — pb.RiskServiceServer
func (s *GRPCServer) GetExposure(ctx context.Context, req *pb.GetExposureRequest) (*pb.GetExposureResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	// Find the COUNTERPARTY limit for the requested BIC.
	l, err := s.svc.Reserve(ctx, tid, domain.LimitCounterparty, req.GetCounterpartyBic(), decimal.Zero)
	if err != nil && !errors.Is(err, domain.ErrInvalidInput) {
		// ErrInvalidInput is expected for the Reserve(zero); we only want the limit projection.
		return nil, mapErr(err)
	}
	if l == nil {
		return nil, status.Error(codes.NotFound, "limit not found")
	}
	utilPct, _ := l.UtilisationPct().Float64()
	return &pb.GetExposureResponse{
		CurrentExposure: &pb.Money{Amount: l.Utilised().String(), Currency: l.Currency()},
		LimitCap:        &pb.Money{Amount: l.Cap().String(), Currency: l.Currency()},
		UtilisationPct:  utilPct,
	}, nil
}

// UpdateLimit — pb.RiskServiceServer
func (s *GRPCServer) UpdateLimit(ctx context.Context, req *pb.UpdateLimitRequest) (*pb.UpdateLimitResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	in := req.GetLimit()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "limit required")
	}
	cap, err := decimal.NewFromString(in.GetCap().GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "cap.amount: %v", err)
	}
	l, err := s.svc.CreateLimit(ctx, domain.NewLimitInput{
		TenantID: tid,
		Type:     domain.LimitType(in.GetType().String()),
		Scope:    in.GetScope(),
		Cap:      cap,
		Currency: in.GetCap().GetCurrency(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.UpdateLimitResponse{Limit: &pb.Limit{
		LimitId:  l.ID().String(),
		TenantId: l.TenantID().String(),
		Type:     in.GetType(),
		Scope:    l.Scope(),
		Cap:      &pb.Money{Amount: l.Cap().String(), Currency: l.Currency()},
		Utilised: &pb.Money{Amount: l.Utilised().String(), Currency: l.Currency()},
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

func mapErr(err error) error {
	switch {
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, application.ErrInvalidInput), errors.Is(err, domain.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrBreached):
		return status.Error(codes.ResourceExhausted, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

// parseLimitType maps the wire value onto a domain LimitType. An unrecognised
// value is rejected rather than defaulted: silently checking the wrong limit is
// worse than refusing the call.
func parseLimitType(v string) (domain.LimitType, error) {
	switch domain.LimitType(strings.ToUpper(strings.TrimSpace(v))) {
	case domain.LimitCounterparty:
		return domain.LimitCounterparty, nil
	case domain.LimitCurrency:
		return domain.LimitCurrency, nil
	case domain.LimitTenor:
		return domain.LimitTenor, nil
	case domain.LimitDV01:
		return domain.LimitDV01, nil
	case domain.LimitVaR:
		return domain.LimitVaR, nil
	case "":
		return "", status.Error(codes.InvalidArgument,
			"limit_type is required (COUNTERPARTY, CURRENCY, TENOR, DV01 or VAR)")
	default:
		return "", status.Errorf(codes.InvalidArgument, "unknown limit_type %q", v)
	}
}
