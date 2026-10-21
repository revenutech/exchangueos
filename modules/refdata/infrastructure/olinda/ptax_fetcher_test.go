package olinda_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/revenutech/exchangeos/modules/refdata/infrastructure/olinda"
)

const happyResponse = `{
  "@odata.context": "...",
  "value": [
    {"cotacaoCompra": 5.1010, "cotacaoVenda": 5.1030, "dataHoraCotacao": "2026-05-22 10:09:35.000", "tipoBoletim": "Abertura"},
    {"cotacaoCompra": 5.1020, "cotacaoVenda": 5.1040, "dataHoraCotacao": "2026-05-22 11:11:24.000", "tipoBoletim": "Intermediário"},
    {"cotacaoCompra": 5.1015, "cotacaoVenda": 5.1035, "dataHoraCotacao": "2026-05-22 12:08:51.000", "tipoBoletim": "Intermediário"},
    {"cotacaoCompra": 5.1025, "cotacaoVenda": 5.1045, "dataHoraCotacao": "2026-05-22 13:12:03.000", "tipoBoletim": "Fechamento PTAX"}
  ]
}`

func newServer(t *testing.T, body string, status int) (*httptest.Server, *olinda.Fetcher) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "dataInicial") {
			t.Errorf("missing dataInicial in query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &olinda.Fetcher{
		BaseURL: srv.URL,
		Client:  srv.Client(),
	}
}

func TestFetcher_HappyPath(t *testing.T) {
	_, f := newServer(t, happyResponse, http.StatusOK)
	ctx := context.Background()
	bd := time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC)

	p, err := f.FetchPTAX(ctx, bd)
	if err != nil {
		t.Fatalf("FetchPTAX: %v", err)
	}
	if !p.Date.Equal(bd) {
		t.Errorf("date: got %v want %v", p.Date, bd)
	}
	wantHours := [4]int{10, 11, 12, 13}
	for i, w := range p.Windows {
		if w.Hour != wantHours[i] {
			t.Errorf("window[%d].hour: got %d want %d", i, w.Hour, wantHours[i])
		}
		if !w.Bid.IsPositive() || !w.Ask.IsPositive() {
			t.Errorf("window[%d] bid/ask not positive: %s / %s", i, w.Bid, w.Ask)
		}
	}

	// Run WeightedFixing through the produced struct.
	wf, err := p.WeightedFixing()
	if err != nil {
		t.Fatalf("WeightedFixing: %v", err)
	}
	if wf.String() != "5.1028" {
		t.Errorf("WeightedFixing: got %s want 5.1028", wf)
	}
}

func TestFetcher_MissingWindow(t *testing.T) {
	// Only 3 windows — missing 12h.
	body := `{"value":[
        {"cotacaoCompra":5.1,"cotacaoVenda":5.11,"dataHoraCotacao":"2026-05-22 10:00:00.000","tipoBoletim":"Abertura"},
        {"cotacaoCompra":5.1,"cotacaoVenda":5.11,"dataHoraCotacao":"2026-05-22 11:00:00.000","tipoBoletim":"Intermediário"},
        {"cotacaoCompra":5.1,"cotacaoVenda":5.11,"dataHoraCotacao":"2026-05-22 13:00:00.000","tipoBoletim":"Fechamento PTAX"}
    ]}`
	_, f := newServer(t, body, http.StatusOK)
	_, err := f.FetchPTAX(context.Background(), time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "missing window hour=12") {
		t.Fatalf("expected missing-window=12 error, got %v", err)
	}
}

func TestFetcher_HTTPError(t *testing.T) {
	_, f := newServer(t, `{"value":[]}`, http.StatusInternalServerError)
	_, err := f.FetchPTAX(context.Background(), time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("expected http 500 error, got %v", err)
	}
}

func TestFetcher_ZeroDate(t *testing.T) {
	f := olinda.New()
	_, err := f.FetchPTAX(context.Background(), time.Time{})
	if err == nil || !strings.Contains(err.Error(), "business_date") {
		t.Fatalf("expected business_date error, got %v", err)
	}
}

func TestFetcher_BadTimestamp(t *testing.T) {
	body := `{"value":[
        {"cotacaoCompra":5.1,"cotacaoVenda":5.11,"dataHoraCotacao":"not-a-time","tipoBoletim":"Abertura"}
    ]}`
	_, f := newServer(t, body, http.StatusOK)
	_, err := f.FetchPTAX(context.Background(), time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC))
	if err == nil || !strings.Contains(err.Error(), "parse ts") {
		t.Fatalf("expected parse ts error, got %v", err)
	}
}

// Smoke test demonstrating the OLINDA URL format the fetcher constructs.
// Ensures the date-quoted parameters are URL-encoded.
func TestFetcher_URLFormat(t *testing.T) {
	var capturedURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		_, _ = w.Write([]byte(`{"value":[]}`))
	}))
	defer srv.Close()
	f := &olinda.Fetcher{BaseURL: srv.URL, Client: srv.Client()}

	_, _ = f.FetchPTAX(context.Background(), time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC))

	for _, want := range []string{
		"CotacaoDolarPeriodo",
		"@dataInicial",
		"@dataFinalCotacao",
		"%2705-22-2026%27", // url-quoted 'MM-DD-YYYY'
		"$format=json",
	} {
		if !strings.Contains(capturedURL, want) {
			t.Errorf("URL missing %q\nfull: %s", want, capturedURL)
		}
	}
}
