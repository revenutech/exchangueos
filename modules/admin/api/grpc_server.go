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

	"github.com/revenutech/exchangeos/modules/admin/application"
	"github.com/revenutech/exchangeos/modules/admin/domain"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

type GRPCServer struct{ svc *application.Service }

func NewGRPCServer(svc *application.Service) *GRPCServer { return &GRPCServer{svc: svc} }

// EmitSystemEvent — pb.AdminServiceServer
func (s *GRPCServer) EmitSystemEvent(ctx context.Context, req *pb.EmitSystemEventRequest) (*pb.EmitSystemEventResponse, error) {
	in := req.GetEvent()
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "event required")
	}
	at := in.GetAt().AsTime()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	e, err := s.svc.EmitSystemEvent(ctx, domain.NewSystemEventInput{
		Code:        domain.EventCode(in.GetCode().String()),
		Component:   in.GetComponent(),
		Description: in.GetDescription(),
		At:          at,
		ISO20022Ref: in.GetIso20022MessageId(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.EmitSystemEventResponse{Event: &pb.SystemEvent{
		EventId:           e.ID().String(),
		Component:         e.Component(),
		Description:       e.Description(),
		At:                timestamppb.New(e.At()),
		Iso20022MessageId: e.ISO20022Ref(),
	}}, nil
}

// GetServiceHealth — pb.AdminServiceServer
//
// Not implemented. It used to answer SERVING unconditionally, for any component
// name, without consulting anything — a health probe that can never report a
// problem is worse than no probe, because monitoring built on it reports green
// through an outage.
func (s *GRPCServer) GetServiceHealth(_ context.Context, _ *pb.GetServiceHealthRequest) (*pb.GetServiceHealthResponse, error) {
	return nil, status.Error(codes.Unimplemented,
		"service health aggregation is not wired; use the HTTP /health endpoint")
}

// TriggerEOD — pb.AdminServiceServer
func (s *GRPCServer) TriggerEOD(ctx context.Context, req *pb.TriggerEODRequest) (*pb.TriggerEODResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	bd := req.GetBusinessDate().AsTime()
	if bd.IsZero() {
		bd = time.Now().UTC()
	}
	j, err := s.svc.TriggerEOD(ctx, tid, bd)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.TriggerEODResponse{
		JobId:     j.ID().String(),
		StartedAt: timestamppb.New(time.Now().UTC()),
	}, nil
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
	case errors.Is(err, application.ErrConflict):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
