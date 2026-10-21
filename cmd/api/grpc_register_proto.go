//go:build grpcgen

package main

import (
	"google.golang.org/grpc"

	"github.com/revenutech/exchangeos/internal/container"
	adminapi "github.com/revenutech/exchangeos/modules/admin/api"
	setlapi "github.com/revenutech/exchangeos/modules/cls_settlement/api"
	complapi "github.com/revenutech/exchangeos/modules/compliance/api"
	posapi "github.com/revenutech/exchangeos/modules/position/api"
	quoteapi "github.com/revenutech/exchangeos/modules/quote/api"
	refapi "github.com/revenutech/exchangeos/modules/refdata/api"
	riskapi "github.com/revenutech/exchangeos/modules/risk/api"
	tradeapi "github.com/revenutech/exchangeos/modules/trade/api"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// registerGeneratedServices wires the gRPC adapter for each bounded context to the
// shared application container. Only compiled when -tags grpcgen is passed AND
// proto/gen/exchangeos/v1 exists.
func registerGeneratedServices(srv *grpc.Server, c *container.Container) {
	pb.RegisterRefDataServiceServer(srv, refapi.NewGRPCServer(c.RefData))
	pb.RegisterQuoteServiceServer(srv, quoteapi.NewGRPCServer(c.Quote))
	pb.RegisterTradeServiceServer(srv, tradeapi.NewGRPCServer(c.Trade))
	pb.RegisterSettlementServiceServer(srv, setlapi.NewGRPCServer(c.Settlement, c.PayIn, c.NetReport))
	pb.RegisterRiskServiceServer(srv, riskapi.NewGRPCServer(c.Risk))
	pb.RegisterPositionServiceServer(srv, posapi.NewGRPCServer(c.Position))
	pb.RegisterComplianceServiceServer(srv, complapi.NewGRPCServer(c.Compliance))
	pb.RegisterAdminServiceServer(srv, adminapi.NewGRPCServer(c.Admin))
	// All 8 bounded-context services are now bound. CFETS Capture + Confirmation
	// have no public proto service yet (intentionally internal to MS-023d2 flow).
}
