package api_test

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	setlapi "github.com/revenutech/exchangeos/modules/cls_settlement/api"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// TestGetNetReport_RefusesUntilDomainCarriesV03Dates pins the MS-079 7.5
// interim behaviour: with valid request fields, the RPC returns
// FailedPrecondition rather than emitting the non-conformant V02 body. The
// test guarantees that merging MS-079 7.5 part 2 (the real V03 emitter) is a
// visible change — a future commit that fixes the renderer will also update
// this test, and anyone replacing part 2 with another V02-style fabrication
// has to delete the assertion deliberately.
func TestGetNetReport_RefusesUntilDomainCarriesV03Dates(t *testing.T) {
	srv := setlapi.NewGRPCServer(nil, nil, nil)

	_, err := srv.GetNetReport(context.Background(), &pb.GetNetReportRequest{
		Tenant: &pb.TenantContext{
			TenantId: "018f7f1e-0000-7000-8000-000000000001",
		},
		CycleId:   "018f7f1e-0000-7000-8000-000000000002",
		MemberBic: "DEUTDEFF",
	})
	if err == nil {
		t.Fatalf("expected FailedPrecondition, got nil error")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if got, want := st.Code(), codes.FailedPrecondition; got != want {
		t.Fatalf("code: got %v want %v", got, want)
	}
	const wantMsg = "camt.088.001.03 pendente: domínio sem NetgCutOffTm/RptDt/ValDt"
	if got := st.Message(); got != wantMsg {
		t.Fatalf("message:\n  got  %q\n  want %q", got, wantMsg)
	}
}

// TestGetNetReport_ValidatesInputsBeforeRefusing ensures the interim block
// does not swallow bad-input errors: a client sending junk still gets
// InvalidArgument, not the FailedPrecondition stand-in.
func TestGetNetReport_ValidatesInputsBeforeRefusing(t *testing.T) {
	srv := setlapi.NewGRPCServer(nil, nil, nil)

	cases := []struct {
		name string
		req  *pb.GetNetReportRequest
		code codes.Code
	}{
		{
			name: "missing tenant",
			req: &pb.GetNetReportRequest{
				CycleId:   "018f7f1e-0000-7000-8000-000000000002",
				MemberBic: "DEUTDEFF",
			},
			code: codes.InvalidArgument,
		},
		{
			name: "cycle_id not uuid",
			req: &pb.GetNetReportRequest{
				Tenant:    &pb.TenantContext{TenantId: "018f7f1e-0000-7000-8000-000000000001"},
				CycleId:   "not-a-uuid",
				MemberBic: "DEUTDEFF",
			},
			code: codes.InvalidArgument,
		},
		{
			name: "member_bic wrong length",
			req: &pb.GetNetReportRequest{
				Tenant:    &pb.TenantContext{TenantId: "018f7f1e-0000-7000-8000-000000000001"},
				CycleId:   "018f7f1e-0000-7000-8000-000000000002",
				MemberBic: "DEUT",
			},
			code: codes.InvalidArgument,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := srv.GetNetReport(context.Background(), tc.req)
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected gRPC status error, got %T: %v", err, err)
			}
			if got := st.Code(); got != tc.code {
				t.Fatalf("code: got %v want %v (msg=%q)", got, tc.code, st.Message())
			}
		})
	}
}
