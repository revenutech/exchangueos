//go:build grpcgen

package main

import (
	"google.golang.org/grpc"

	"github.com/revenutech/exchangeos/internal/container"
	adminapi "github.com/revenutech/exchangeos/modules/admin/api"
	complapi "github.com/revenutech/exchangeos/modules/compliance/api"
	posapi "github.com/revenutech/exchangeos/modules/position/api"
	quoteapi "github.com/revenutech/exchangeos/modules/quote/api"
	refapi "github.com/revenutech/exchangeos/modules/refdata/api"
	riskapi "github.com/revenutech/exchangeos/modules/risk/api"
	tradeapi "github.com/revenutech/exchangeos/modules/trade/api"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// registerGeneratedServices wires the gRPC adapter for each still-gated
// bounded context. Only compiled when -tags grpcgen is passed AND
// proto/gen/exchangeos/v1 exists. SettlementService was promoted to the
// default build in MS-079 7.5 (grpc_register_settlement.go) — it is NOT
// registered here, to avoid double-binding under -tags grpcgen.
func registerGeneratedServices(srv *grpc.Server, c *container.Container) {
	pb.RegisterRefDataServiceServer(srv, refapi.NewGRPCServer(c.RefData))
	pb.RegisterQuoteServiceServer(srv, quoteapi.NewGRPCServer(c.Quote))
	pb.RegisterTradeServiceServer(srv, tradeapi.NewGRPCServer(c.Trade))
	pb.RegisterRiskServiceServer(srv, riskapi.NewGRPCServer(c.Risk))
	pb.RegisterPositionServiceServer(srv, posapi.NewGRPCServer(c.Position))
	pb.RegisterComplianceServiceServer(srv, complapi.NewGRPCServer(c.Compliance))
	pb.RegisterAdminServiceServer(srv, adminapi.NewGRPCServer(c.Admin))
	// Seven BC services remain here. SettlementService lives in
	// grpc_register_settlement.go (unconditional). CFETS Capture + Confirmation
	// have no public proto service yet (intentionally internal to MS-023d2 flow).
}
