package api_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	setlapi "github.com/revenutech/exchangeos/modules/cls_settlement/api"
	clsapp "github.com/revenutech/exchangeos/modules/cls_settlement/application"
	clsmem "github.com/revenutech/exchangeos/modules/cls_settlement/infrastructure/memory"
	netapp "github.com/revenutech/exchangeos/modules/netreport/application"
	netdomain "github.com/revenutech/exchangeos/modules/netreport/domain"
	netmem "github.com/revenutech/exchangeos/modules/netreport/infrastructure/memory"
	pb "github.com/revenutech/exchangeos/proto/gen/exchangeos/v1"
)

// seedNetReportFixture opens a cycle and generates two per-currency NetReports
// — one receivable (EUR), one payable (USD) — then returns the wired server
// plus the tenant/cycle IDs the test asserts against.
func seedNetReportFixture(t *testing.T) (*setlapi.GRPCServer, uuid.UUID, uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tenant := uuid.New()

	cycleRepo := clsmem.NewCycleRepo()
	cyclePub := clsmem.NewNoopPublisher()
	cycles := clsapp.NewService(cycleRepo, cyclePub)

	netRepo := netmem.NewRepo()
	nets := netapp.NewService(netRepo)

	cycleDate := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	cycle, err := cycles.OpenCycle(ctx, tenant, cycleDate)
	if err != nil {
		t.Fatalf("OpenCycle: %v", err)
	}

	if _, err := nets.Generate(ctx, netdomain.NewNetReportInput{
		TenantID:    tenant,
		CycleID:     cycle.ID(),
		Currency:    "EUR",
		GrossPayIn:  decimal.RequireFromString("1000000"),
		GrossPayOut: decimal.RequireFromString("750000"),
		TradeCount:  3,
		GeneratedAt: time.Date(2026, 5, 26, 10, 15, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Generate EUR: %v", err)
	}
	if _, err := nets.Generate(ctx, netdomain.NewNetReportInput{
		TenantID:    tenant,
		CycleID:     cycle.ID(),
		Currency:    "USD",
		GrossPayIn:  decimal.RequireFromString("500000"),
		GrossPayOut: decimal.RequireFromString("770123.45"),
		TradeCount:  2,
		GeneratedAt: time.Date(2026, 5, 26, 10, 15, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("Generate USD: %v", err)
	}

	return setlapi.NewGRPCServer(cycles, nil, nets), tenant, cycle.ID()
}

// TestGetNetReport_EmitsCamt088V03 covers the MS-079 7.5 part 2 happy path:
// GetNetReport joins the cycle (PIN3 → NetgCutOffTm, CycleDate → RptDt/ValDt)
// with per-currency NetReports and emits a conformant camt.088.001.03 body.
func TestGetNetReport_EmitsCamt088V03(t *testing.T) {
	srv, tenant, cycleID := seedNetReportFixture(t)

	resp, err := srv.GetNetReport(context.Background(), &pb.GetNetReportRequest{
		Tenant:    &pb.TenantContext{TenantId: tenant.String()},
		CycleId:   cycleID.String(),
		MemberBic: "DEUTDEFF",
	})
	if err != nil {
		t.Fatalf("GetNetReport: %v", err)
	}
	body := resp.GetReportXml()

	// Namespace and envelope anchor — a V02 fabrication would still carry
	// a <NetRpt> root but the namespace label pins which message this is.
	if !strings.Contains(body, `xmlns="urn:iso:std:iso:20022:tech:xsd:camt.088.001.03"`) {
		t.Fatalf("missing camt.088.001.03 namespace in body:\n%s", body)
	}

	// The three domain-sourced ISO fields — the ones whose absence drove
	// MS-079 7.5 — are now carried from CLSCycle.
	for _, want := range []string{
		`<RptDt>2026-05-26</RptDt>`,
		`<ValDt>2026-05-26</ValDt>`,
		`<NetgCutOffTm>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in body:\n%s", want, body)
		}
	}

	// Member BIC under <NetSvcPtcptId> (PartyIdentification242Choice.AnyBIC).
	if !strings.Contains(body, `<NetSvcPtcptId>`) || !strings.Contains(body, `>DEUTDEFF<`) {
		t.Errorf("member BIC not emitted under <NetSvcPtcptId>:\n%s", body)
	}
	// CLS Bank BIC under <NetSvcCtrPtyId>.
	if !strings.Contains(body, `<NetSvcCtrPtyId>`) || !strings.Contains(body, `>CLSBUS33<`) {
		t.Errorf("CLS BIC not emitted under <NetSvcCtrPtyId>:\n%s", body)
	}

	// EUR (payable in = 1_000_000, out = 750_000 → +250_000 → RECE)
	// USD (payable in = 500_000, out = 770_123.45 → -270_123.45 → PAYM, abs value)
	for _, want := range []string{
		`<Amt Ccy="EUR">250000</Amt>`,
		`<OblgtnDrctn>RECE</OblgtnDrctn>`,
		`<Amt Ccy="USD">270123.45</Amt>`,
		`<OblgtnDrctn>PAYM</OblgtnDrctn>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in body:\n%s", want, body)
		}
	}

	// generated_at is populated from the first NetReport.GeneratedAt — not
	// zero, not time.Now; the test relies on the fixture's deterministic value.
	if got := resp.GetGeneratedAt().AsTime().UTC(); !got.Equal(time.Date(2026, 5, 26, 10, 15, 0, 0, time.UTC)) {
		t.Errorf("generated_at: got %v want 2026-05-26T10:15:00Z", got)
	}
}

// TestGetNetReport_ValidatesInputsBeforeLoading pins the gRPC error contract:
// bad input never touches the application layer.
func TestGetNetReport_ValidatesInputsBeforeLoading(t *testing.T) {
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
				Tenant:    &pb.TenantContext{TenantId: uuid.New().String()},
				CycleId:   "not-a-uuid",
				MemberBic: "DEUTDEFF",
			},
			code: codes.InvalidArgument,
		},
		{
			name: "member_bic wrong length",
			req: &pb.GetNetReportRequest{
				Tenant:    &pb.TenantContext{TenantId: uuid.New().String()},
				CycleId:   uuid.New().String(),
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

// TestGetNetReport_UnknownCycleReturnsNotFound guards the end-to-end 404 path.
func TestGetNetReport_UnknownCycleReturnsNotFound(t *testing.T) {
	srv, tenant, _ := seedNetReportFixture(t)

	_, err := srv.GetNetReport(context.Background(), &pb.GetNetReportRequest{
		Tenant:    &pb.TenantContext{TenantId: tenant.String()},
		CycleId:   uuid.New().String(), // different cycle
		MemberBic: "DEUTDEFF",
	})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T: %v", err, err)
	}
	if got := st.Code(); got != codes.NotFound {
		t.Fatalf("code: got %v want NotFound (msg=%q)", got, st.Message())
	}
}
