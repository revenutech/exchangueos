//go:build !grpcgen

package main

import (
	"google.golang.org/grpc"

	"github.com/revenutech/exchangeos/internal/container"
)

// registerGeneratedServices is a no-op when proto/gen has not been produced.
// To enable, run `task proto:gen` and build with `-tags grpcgen`.
func registerGeneratedServices(_ *grpc.Server, _ *container.Container) {
	// no-op
}
