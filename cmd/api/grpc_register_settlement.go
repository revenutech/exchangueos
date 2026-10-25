package main

import (
	"google.golang.org/grpc"

	"github.com/revenutech/exchangeos/internal/container"
	setlapi "github.com/revenutech/exchangeos/modules/cls_settlement/api"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// registerSettlementService binds the cls_settlement gRPC adapter in every
// build. Entered the default build in MS-079 7.5 so camt.088 NetReport has a
// real production path. The other seven bounded-context adapters stay behind
// -tags grpcgen until their own milestones deliver real callers.
//
// Requires `task proto:gen` to have produced proto/gen/exchangeos/v1/*.pb.go.
func registerSettlementService(srv *grpc.Server, c *container.Container) {
	pb.RegisterSettlementServiceServer(srv, setlapi.NewGRPCServer(c.Settlement, c.PayIn, c.NetReport))
}
