// Package olinda — concrete pricing.PTAXFetcher backed by the BACEN OLINDA REST API.
//
// Reference:
//
//	https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata
//
// Endpoint used: CotacaoDolarPeriodo (Cotação Dólar por Período). Filters a
// date range, returns 4 windows per business date (Abertura, two
// Intermediários, Fechamento PTAX).
package olinda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/revenutech/exchangeos/pkg/pricing"
)

// DefaultBaseURL is the public OLINDA endpoint. Override in tests.
const DefaultBaseURL = "https://olinda.bcb.gov.br/olinda/servico/PTAX/versao/v1/odata"

// Fetcher implements pricing.PTAXFetcher.
type Fetcher struct {
	BaseURL string
	Client  *http.Client
}

// New returns a fetcher with sensible defaults (10s timeout).
func New() *Fetcher {
	return &Fetcher{
		BaseURL: DefaultBaseURL,
		Client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// FetchPTAX implements pricing.PTAXFetcher.
func (f *Fetcher) FetchPTAX(ctx context.Context, businessDate time.Time) (pricing.PTAX, error) {
	bd := businessDate.UTC()
	if bd.IsZero() {
		return pricing.PTAX{}, fmt.Errorf("olinda: business_date required")
	}
	day := bd.Format("01-02-2006") // OLINDA uses MM-DD-YYYY

	// CotacaoDolarPeriodo(dataInicial='MM-DD-YYYY',dataFinalCotacao='MM-DD-YYYY')?...
	rel := fmt.Sprintf(
		"/CotacaoDolarPeriodo(dataInicial=@dataInicial,dataFinalCotacao=@dataFinalCotacao)"+
			"?@dataInicial=%s&@dataFinalCotacao=%s&$top=20&$format=json",
		url.QueryEscape("'"+day+"'"), url.QueryEscape("'"+day+"'"),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.BaseURL+rel, nil)
	if err != nil {
		return pricing.PTAX{}, fmt.Errorf("olinda: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.Client.Do(req)
	if err != nil {
		return pricing.PTAX{}, fmt.Errorf("olinda: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return pricing.PTAX{}, fmt.Errorf("olinda: http %d", resp.StatusCode)
	}

	var body struct {
		Value []struct {
			CotacaoCompra   decimal.Decimal `json:"cotacaoCompra"`
			CotacaoVenda    decimal.Decimal `json:"cotacaoVenda"`
			DataHoraCotacao string          `json:"dataHoraCotacao"` // "YYYY-MM-DD HH:MM:SS.SSS"
			TipoBoletim     string          `json:"tipoBoletim"`     // Abertura | Intermediário | Fechamento PTAX
		} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return pricing.PTAX{}, fmt.Errorf("olinda: decode: %w", err)
	}

	return buildPTAX(bd, body.Value)
}

// buildPTAX maps OLINDA rows into a PTAX struct, choosing one row per window.
// The 4 BACEN windows correspond to São Paulo hours 10/11/12/13. We map by
// timestamp hour: 10 → window[0], 11 → window[1], 12 → window[2], 13 → window[3].
func buildPTAX(businessDate time.Time, rows []struct {
	CotacaoCompra   decimal.Decimal `json:"cotacaoCompra"`
	CotacaoVenda    decimal.Decimal `json:"cotacaoVenda"`
	DataHoraCotacao string          `json:"dataHoraCotacao"`
	TipoBoletim     string          `json:"tipoBoletim"`
}) (pricing.PTAX, error) {
	wantHours := [4]int{10, 11, 12, 13}
	got := [4]*pricing.PTAXWindow{}

	for _, r := range rows {
		ts, err := parseOlindaTS(r.DataHoraCotacao)
		if err != nil {
			return pricing.PTAX{}, fmt.Errorf("olinda: parse ts %q: %w", r.DataHoraCotacao, err)
		}
		hour := ts.Hour()
		for i, h := range wantHours {
			if hour == h {
				got[i] = &pricing.PTAXWindow{
					Hour: h,
					Bid:  r.CotacaoCompra,
					Ask:  r.CotacaoVenda,
				}
			}
		}
	}

	var windows [4]pricing.PTAXWindow
	for i, w := range got {
		if w == nil {
			return pricing.PTAX{}, fmt.Errorf("olinda: missing window hour=%d", wantHours[i])
		}
		windows[i] = *w
	}
	return pricing.PTAX{
		Date:    time.Date(businessDate.Year(), businessDate.Month(), businessDate.Day(), 0, 0, 0, 0, time.UTC),
		Windows: windows,
	}, nil
}

// parseOlindaTS accepts OLINDA's "YYYY-MM-DD HH:MM:SS(.fff)?" format.
func parseOlindaTS(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range []string{
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported timestamp layout: %q", s)
}
