//go:build grpcgen

package api

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/revenutech/exchangeos/modules/trade/application"
	"github.com/revenutech/exchangeos/modules/trade/domain"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

type GRPCServer struct {
	svc *application.Service
}

func NewGRPCServer(svc *application.Service) *GRPCServer { return &GRPCServer{svc: svc} }

// CreateTrade — pb.TradeServiceServer
func (s *GRPCServer) CreateTrade(ctx context.Context, req *pb.CreateTradeRequest) (*pb.CreateTradeResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	pt := req.GetTrade()
	if pt == nil {
		return nil, status.Error(codes.InvalidArgument, "trade required")
	}
	boughtAmt, err := decimal.NewFromString(pt.GetBoughtAmount().GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bought_amount: %v", err)
	}
	soldAmt, err := decimal.NewFromString(pt.GetSoldAmount().GetAmount())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "sold_amount: %v", err)
	}
	rate, err := decimal.NewFromString(pt.GetDealRate().GetRate())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "deal_rate: %v", err)
	}

	t, err := s.svc.BookTrade(ctx, application.BookTradeRequest{
		TenantID:       tid,
		ExternalRef:    pt.GetExternalRef(),
		Type:           domain.TradeType(pt.GetType().String()),
		Venue:          domain.SettlementVenue(pt.GetVenue().String()),
		BuyerBIC:       pt.GetBuyer().GetBic(),
		SellerBIC:      pt.GetSeller().GetBic(),
		BoughtCurrency: pt.GetBoughtAmount().GetCurrency(),
		BoughtAmount:   boughtAmt,
		SoldCurrency:   pt.GetSoldAmount().GetCurrency(),
		SoldAmount:     soldAmt,
		DealRate:       rate,
		TradeDate:      pt.GetTradeDate().AsTime(),
		ValueDate:      pt.GetValueDate().AsTime(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CreateTradeResponse{Trade: toPB(t)}, nil
}

// GetTrade — pb.TradeServiceServer
func (s *GRPCServer) GetTrade(ctx context.Context, req *pb.GetTradeRequest) (*pb.GetTradeResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetTradeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trade_id: %v", err)
	}
	t, err := s.svc.GetTrade(ctx, id)
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.GetTradeResponse{Trade: toPB(t)}, nil
}

// ListTrades — pb.TradeServiceServer
func (s *GRPCServer) ListTrades(ctx context.Context, req *pb.ListTradesRequest) (*pb.ListTradesResponse, error) {
	tid, err := parseTenant(req.GetTenant())
	if err != nil {
		return nil, err
	}
	limit := 100
	if req.GetPage() != nil && req.GetPage().GetPageSize() > 0 {
		limit = int(req.GetPage().GetPageSize())
	}
	var from, to = req.GetFrom().AsTime(), req.GetTo().AsTime()
	list, err := s.svc.ListTrades(ctx, tid,
		domain.TradeStatus(req.GetStatusFilter().String()),
		from, to, limit,
	)
	if err != nil {
		return nil, mapErr(err)
	}
	out := make([]*pb.Trade, 0, len(list))
	for _, t := range list {
		out = append(out, toPB(t))
	}
	return &pb.ListTradesResponse{Trades: out}, nil
}

// CancelTrade — pb.TradeServiceServer
func (s *GRPCServer) CancelTrade(ctx context.Context, req *pb.CancelTradeRequest) (*pb.CancelTradeResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetTradeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trade_id: %v", err)
	}
	t, err := s.svc.CancelTrade(ctx, id, req.GetReason())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.CancelTradeResponse{Trade: toPB(t)}, nil
}

// SettleTrade — pb.TradeServiceServer (transitions CONFIRMED → SETTLING → SETTLED in one call).
func (s *GRPCServer) SettleTrade(ctx context.Context, req *pb.SettleTradeRequest) (*pb.SettleTradeResponse, error) {
	if _, err := parseTenant(req.GetTenant()); err != nil {
		return nil, err
	}
	id, err := uuid.Parse(req.GetTradeId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "trade_id: %v", err)
	}
	if _, err := s.svc.MarkSettling(ctx, id); err != nil {
		return nil, mapErr(err)
	}
	t, err := s.svc.MarkSettled(ctx, id, req.GetSettlementRef())
	if err != nil {
		return nil, mapErr(err)
	}
	return &pb.SettleTradeResponse{Trade: toPB(t)}, nil
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

func toPB(t *domain.FXTrade) *pb.Trade {
	return &pb.Trade{
		TradeId:     t.ID().String(),
		TenantId:    t.TenantID().String(),
		ExternalRef: t.ExternalRef(),
		Type:        pb.TradeType(pb.TradeType_value["TRADE_TYPE_"+string(t.Type())]),
		Status:      pb.TradeStatus(pb.TradeStatus_value["TRADE_STATUS_"+string(t.Status())]),
		Venue:       pb.SettlementVenue(pb.SettlementVenue_value["SETTLEMENT_VENUE_"+string(t.Venue())]),
		Buyer:       &pb.Party{Bic: t.BuyerBIC()},
		Seller:      &pb.Party{Bic: t.SellerBIC()},
		BoughtAmount: &pb.Money{Amount: t.BoughtAmount().String(), Currency: t.BoughtCurrency()},
		SoldAmount:   &pb.Money{Amount: t.SoldAmount().String(), Currency: t.SoldCurrency()},
		DealRate:     &pb.FxRate{Rate: t.DealRate().String(), BaseCcy: t.BoughtCurrency(), QuoteCcy: t.SoldCurrency()},
		TradeDate:    timestamppb.New(t.TradeDate()),
		ValueDate:    timestamppb.New(t.ValueDate()),
	}
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, application.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, application.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrCancelReasonRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrInvalidTransition):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
