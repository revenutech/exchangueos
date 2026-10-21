package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/modules/position/domain"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestNewPosition_Happy(t *testing.T) {
	p, err := domain.NewPosition(uuid.New(), "usd")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if p.Currency() != "USD" {
		t.Errorf("ccy: %s", p.Currency())
	}
	if !p.IsFlat() {
		t.Errorf("expected flat at start, got long=%s short=%s", p.Long(), p.Short())
	}
}

func TestNewPosition_BadInputs(t *testing.T) {
	if _, err := domain.NewPosition(uuid.Nil, "USD"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("nil tenant: %v", err)
	}
	if _, err := domain.NewPosition(uuid.New(), "DOLLAR"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bad ccy: %v", err)
	}
}

func TestApplyTradeLeg_BuyIncreasesLong(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	if err := p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideBuy, Amount: dec("1000000")}); err != nil {
		t.Fatalf("ApplyTradeLeg: %v", err)
	}
	if !p.Long().Equal(dec("1000000")) || !p.Net().Equal(dec("1000000")) || !p.IsLong() {
		t.Fatalf("long=%s net=%s isLong=%v", p.Long(), p.Net(), p.IsLong())
	}
}

func TestApplyTradeLeg_SellIncreasesShort(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	if err := p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideSell, Amount: dec("500000")}); err != nil {
		t.Fatalf("ApplyTradeLeg: %v", err)
	}
	if !p.Short().Equal(dec("500000")) || !p.Net().Equal(dec("-500000")) || !p.IsShort() {
		t.Fatalf("short=%s net=%s isShort=%v", p.Short(), p.Net(), p.IsShort())
	}
}

func TestApplyTradeLeg_NetsCleanly(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	_ = p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideBuy, Amount: dec("1000000")})
	_ = p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideSell, Amount: dec("1000000")})
	if !p.IsFlat() {
		t.Fatalf("expected flat after offsetting trades")
	}
}

func TestApplyTradeLeg_BadInputs(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	if err := p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideBuy, Amount: dec("0")}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("zero amt: %v", err)
	}
	if err := p.ApplyTradeLeg(domain.TradeLeg{Side: "BOGUS", Amount: dec("1")}); !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("bad side: %v", err)
	}
}

func TestApplyTradeLeg_DefaultsAsOf(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	before := p.AsOf()
	time.Sleep(1 * time.Millisecond)
	_ = p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideBuy, Amount: dec("1")})
	if !p.AsOf().After(before) {
		t.Fatalf("AsOf not advanced")
	}
}

func TestVersion_Increments(t *testing.T) {
	p, _ := domain.NewPosition(uuid.New(), "USD")
	v0 := p.Version()
	_ = p.ApplyTradeLeg(domain.TradeLeg{Side: domain.SideBuy, Amount: dec("1")})
	if p.Version() != v0+1 {
		t.Fatalf("version: got %d want %d", p.Version(), v0+1)
	}
}
