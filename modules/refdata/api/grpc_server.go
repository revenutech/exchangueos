//go:build grpcgen

package api

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/revenutech/exchangeos/modules/refdata/application"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// GRPCServer adapts pb.RefDataServiceServer to the application service.
type GRPCServer struct {
	svc *application.Service
}

func NewGRPCServer(svc *application.Service) *GRPCServer {
	return &GRPCServer{svc: svc}
}

// ListCurrencies — pb.RefDataServiceServer
func (s *GRPCServer) ListCurrencies(ctx context.Context, _ *pb.ListCurrenciesRequest) (*pb.ListCurrenciesResponse, error) {
	list, err := s.svc.ListCurrencies(ctx, true)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*pb.Currency, 0, len(list))
	for _, c := range list {
		out = append(out, &pb.Currency{
			Code:           c.Code(),
			Name:           c.Name(),
			MinorUnits:     int32(c.MinorUnits()),
			ClsEligible:    c.IsCLSEligible(),
			CfetsEligible:  c.IsCFETSEligible(),
		})
	}
	return &pb.ListCurrenciesResponse{Currencies: out}, nil
}

// GetCalendar — pb.RefDataServiceServer
func (s *GRPCServer) GetCalendar(ctx context.Context, req *pb.GetCalendarRequest) (*pb.GetCalendarResponse, error) {
	cal, err := s.svc.GetCalendar(ctx, req.GetCalendarId())
	if err != nil {
		return nil, mapErr(err)
	}
	hols := cal.HolidaysSorted()
	pbHols := make([]*timestamppb.Timestamp, 0, len(hols))
	for _, h := range hols {
		pbHols = append(pbHols, timestamppb.New(h))
	}
	return &pb.GetCalendarResponse{
		Calendar: &pb.Calendar{
			CalendarId: cal.ID(),
			Holidays:   pbHols,
		},
	}, nil
}

// ResolveBIC — pb.RefDataServiceServer
func (s *GRPCServer) ResolveBIC(ctx context.Context, req *pb.ResolveBICRequest) (*pb.ResolveBICResponse, error) {
	b, err := s.svc.ResolveBIC(ctx, req.GetBic())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.ResolveBICResponse{
		BicRecord: &pb.BICRecord{
			Bic:             b.BIC(),
			InstitutionName: b.InstitutionName(),
			Country:         b.Country(),
			Lei:             b.LEI(),
			Active:          b.IsActive(),
		},
	}, nil
}

// GetSSI — pb.RefDataServiceServer
func (s *GRPCServer) GetSSI(ctx context.Context, req *pb.GetSSIRequest) (*pb.GetSSIResponse, error) {
	if req.GetTenant() == nil {
		return nil, status.Error(codes.InvalidArgument, "tenant context required")
	}
	tid, err := uuid.Parse(req.GetTenant().GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "tenant_id: %v", err)
	}
	var at time.Time
	if req.GetAtTime() != nil {
		at = req.GetAtTime().AsTime()
	}
	ssi, err := s.svc.GetSSI(ctx, tid, req.GetCounterpartyBic(), req.GetCurrency(), at)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.GetSSIResponse{
		Ssi: &pb.SSI{
			SsiId:           ssi.ID().String(),
			TenantId:        ssi.TenantID().String(),
			Currency:        ssi.Currency(),
			BeneficiaryBic:  ssi.BeneficiaryBIC(),
			IntermediaryBic: ssi.IntermediaryBIC(),
			AccountNumber:   ssi.AccountNumber(),
			Iban:            ssi.IBAN(),
		},
	}, nil
}

// mapErr converts application sentinels to canonical gRPC codes.
func mapErr(err error) error {
	switch {
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, application.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
