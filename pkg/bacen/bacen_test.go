package bacen_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/bacen"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

// ─── Classifier ───────────────────────────────────────────────────────────

func TestClassifier_ByCode(t *testing.T) {
	c := bacen.NewClassifier()
	// 10000 is the canonical export code from the generated catalogue.
	n, ok := c.ByCode("10000")
	if !ok {
		t.Fatal("expected 10000 in catalog")
	}
	if n.Nature != bacen.NatureIngresso {
		t.Errorf("nature: got %s want INGRESSO", n.Nature)
	}
	// Legacy builtin codes still resolve for backward compat.
	if _, ok := c.ByCode("60001"); !ok {
		t.Fatal("legacy builtin 60001 must still resolve")
	}
}

func TestClassifier_ByCode_Missing(t *testing.T) {
	c := bacen.NewClassifier()
	if _, ok := c.ByCode("99999999"); ok {
		t.Fatal("expected miss")
	}
}

func TestClassifier_Classify_Hints(t *testing.T) {
	c := bacen.NewClassifier()
	cases := []struct {
		hint     string
		wantCode string
	}{
		{"Exportação de café para Itália", "10000"},
		{"Importação de máquinas da Alemanha", "10200"},
		{"Pagamento de royalty de patente", "60200"},
		{"Aporte de investimento estrangeiro direto", "20000"},
		{"Compra de moeda para viagem turística", "40001"},
		{"Compras com cartão de crédito internacional", "50000"},
		{"Conversão entre moedas estrangeiras EUR/JPY", "80000"},
		{"Operação Forward de moeda estrangeira", "70001"},
	}
	for _, tc := range cases {
		t.Run(tc.hint, func(t *testing.T) {
			n, err := c.Classify(tc.hint)
			if err != nil {
				t.Fatalf("Classify(%q): %v", tc.hint, err)
			}
			if n.Code != tc.wantCode {
				t.Errorf("got %s want %s", n.Code, tc.wantCode)
			}
		})
	}
}

func TestClassifier_Classify_Unknown(t *testing.T) {
	c := bacen.NewClassifier()
	if _, err := c.Classify(""); !errors.Is(err, bacen.ErrUnknown) {
		t.Errorf("empty hint: %v", err)
	}
	if _, err := c.Classify("xyzzy not a real keyword"); !errors.Is(err, bacen.ErrUnknown) {
		t.Errorf("nonsense hint: %v", err)
	}
}

func TestClassifier_AllReturnsBuiltinSet(t *testing.T) {
	c := bacen.NewClassifier()
	if got := len(c.All()); got < 15 {
		t.Fatalf("All count: got %d want >= 15", got)
	}
}

// ─── IOF Calculator ───────────────────────────────────────────────────────

func TestIOF_DefaultRate(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	// USD 10,000 × 0.38% = USD 38.00
	rate, amt, err := calc.Compute("DEFAULT", decimal.NewFromInt(10000))
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !rate.Equal(dec("0.0038")) {
		t.Errorf("rate: %s", rate)
	}
	if !amt.Equal(dec("38.00")) {
		t.Errorf("amount: %s", amt)
	}
}

func TestIOF_TravelCash(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	rate, amt, err := calc.Compute("TRAVEL_CASH", decimal.NewFromInt(5000))
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !rate.Equal(dec("0.011")) {
		t.Errorf("rate: %s", rate)
	}
	if !amt.Equal(dec("55.00")) {
		t.Errorf("amount: %s", amt)
	}
}

func TestIOF_ExportZero(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	_, amt, err := calc.Compute("EXPORT", decimal.NewFromInt(1_000_000))
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !amt.IsZero() {
		t.Errorf("export should be zero, got %s", amt)
	}
}

func TestIOF_LoanRate(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	rate, amt, err := calc.Compute("LOAN_SHORT", decimal.NewFromInt(100000))
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if !rate.Equal(dec("0.0638")) {
		t.Errorf("rate: %s", rate)
	}
	// 100,000 × 0.0638 = 6,380.00
	if !amt.Equal(dec("6380.00")) {
		t.Errorf("amount: %s", amt)
	}
}

func TestIOF_BadOperation(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	if _, _, err := calc.Compute("BOGUS", decimal.NewFromInt(100)); !errors.Is(err, bacen.ErrUnknown) {
		t.Fatalf("want ErrUnknown, got %v", err)
	}
}

func TestIOF_NonPositiveNotional(t *testing.T) {
	calc := bacen.NewIOFCalculator()
	if _, _, err := calc.Compute("DEFAULT", decimal.Zero); err == nil {
		t.Fatal("expected error on zero notional")
	}
	if _, _, err := calc.Compute("DEFAULT", dec("-1")); err == nil {
		t.Fatal("expected error on negative notional")
	}
}

func TestIOF_ExtraRatesOverride(t *testing.T) {
	custom := map[string]decimal.Decimal{
		"CUSTOM_OP": dec("0.0500"),
		"DEFAULT":   dec("0.0010"), // override
	}
	calc := bacen.NewIOFCalculator(custom)
	r, _, _ := calc.Compute("CUSTOM_OP", decimal.NewFromInt(100))
	if !r.Equal(dec("0.05")) {
		t.Errorf("CUSTOM_OP rate: %s", r)
	}
	r, _, _ = calc.Compute("DEFAULT", decimal.NewFromInt(100))
	if !r.Equal(dec("0.001")) {
		t.Errorf("overridden DEFAULT: %s", r)
	}
}
