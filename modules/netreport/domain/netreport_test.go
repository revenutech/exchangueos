package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/netreport/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestNetReport_Receivable(t *testing.T) {
	n, err := domain.NewNetReport(domain.NewNetReportInput{
		TenantID:    uuid.New(),
		CycleID:     uuid.New(),
		Currency:    "usd",
		GrossPayIn:  dec("1000000"),
		GrossPayOut: dec("750000"),
		TradeCount:  3,
	})
	if err != nil {
		t.Fatalf("NewNetReport: %v", err)
	}
	if n.Currency() != "USD" {
		t.Errorf("ccy: %s", n.Currency())
	}
	if !n.NetSettlement().Equal(dec("250000")) {
		t.Errorf("net: got %s want 250000", n.NetSettlement())
	}
	if !n.IsReceivable() || n.IsPayable() {
		t.Errorf("receivable flags: rec=%v pay=%v", n.IsReceivable(), n.IsPayable())
	}
}

func TestNetReport_Payable(t *testing.T) {
	n, err := domain.NewNetReport(domain.NewNetReportInput{
		TenantID:    uuid.New(),
		CycleID:     uuid.New(),
		Currency:    "EUR",
		GrossPayIn:  dec("500000"),
		GrossPayOut: dec("800000"),
		TradeCount:  2,
	})
	if err != nil {
		t.Fatalf("NewNetReport: %v", err)
	}
	if !n.NetSettlement().Equal(dec("-300000")) {
		t.Errorf("net: got %s want -300000", n.NetSettlement())
	}
	if n.IsReceivable() || !n.IsPayable() {
		t.Errorf("flags wrong: rec=%v pay=%v", n.IsReceivable(), n.IsPayable())
	}
}

func TestNetReport_Flat(t *testing.T) {
	n, _ := domain.NewNetReport(domain.NewNetReportInput{
		TenantID:    uuid.New(),
		CycleID:     uuid.New(),
		Currency:    "GBP",
		GrossPayIn:  dec("1000"),
		GrossPayOut: dec("1000"),
		TradeCount:  1,
	})
	if !n.NetSettlement().IsZero() {
		t.Errorf("expected zero net, got %s", n.NetSettlement())
	}
	if n.IsReceivable() || n.IsPayable() {
		t.Errorf("flat should be neither receivable nor payable")
	}
}

func TestNetReport_BadInputs(t *testing.T) {
	cases := []struct {
		name string
		in   domain.NewNetReportInput
	}{
		{"nil tenant", domain.NewNetReportInput{CycleID: uuid.New(), Currency: "USD", GrossPayIn: dec("1"), GrossPayOut: dec("1")}},
		{"nil cycle", domain.NewNetReportInput{TenantID: uuid.New(), Currency: "USD", GrossPayIn: dec("1"), GrossPayOut: dec("1")}},
		{"bad ccy", domain.NewNetReportInput{TenantID: uuid.New(), CycleID: uuid.New(), Currency: "DOLLAR", GrossPayIn: dec("1"), GrossPayOut: dec("1")}},
		{"negative in", domain.NewNetReportInput{TenantID: uuid.New(), CycleID: uuid.New(), Currency: "USD", GrossPayIn: dec("-1"), GrossPayOut: dec("1")}},
		{"negative out", domain.NewNetReportInput{TenantID: uuid.New(), CycleID: uuid.New(), Currency: "USD", GrossPayIn: dec("1"), GrossPayOut: dec("-1")}},
		{"negative count", domain.NewNetReportInput{TenantID: uuid.New(), CycleID: uuid.New(), Currency: "USD", GrossPayIn: dec("1"), GrossPayOut: dec("1"), TradeCount: -1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := domain.NewNetReport(tc.in)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Fatalf("want ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestNetReport_GeneratedAtDefaults(t *testing.T) {
	n, _ := domain.NewNetReport(domain.NewNetReportInput{
		TenantID: uuid.New(), CycleID: uuid.New(), Currency: "USD",
		GrossPayIn: dec("1"), GrossPayOut: dec("1"),
		GeneratedAt: time.Time{},
	})
	if n.GeneratedAt().IsZero() {
		t.Fatal("expected GeneratedAt to default to now")
	}
}
