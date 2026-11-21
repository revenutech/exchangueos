package bacen

//go:generate go run ./codegen --input ../../data/bacen/nature-codes-circ-3690-v20260101.csv --output ./codes_full.go

import (
	"fmt"
	"strings"
)

// Nature mirrors compliance.domain.Nature without an import cycle.
type Nature string

const (
	NatureRemessa   Nature = "REMESSA"
	NatureIngresso  Nature = "INGRESSO"
	NatureConversao Nature = "CONVERSAO"
)

// NatureCode is one row of the BACEN catalog.
type NatureCode struct {
	Code        string
	Description string
	Nature      Nature
}

// builtin is the seed set (~20 most-common codes). Production deployments load
// the full 95-code list from refdata at boot; this seed is sufficient for tests
// + smoke runs.
var builtin = []NatureCode{
	// Mercadorias (goods)
	{"10001", "Exportação de mercadorias", NatureIngresso},
	{"10002", "Importação de mercadorias", NatureRemessa},
	{"10005", "Devolução de exportação", NatureRemessa},
	{"10006", "Devolução de importação", NatureIngresso},

	// Serviços (services)
	{"20001", "Receita de serviços técnicos", NatureIngresso},
	{"20002", "Pagamento de serviços técnicos", NatureRemessa},
	{"20010", "Royalties — recebimento", NatureIngresso},
	{"20011", "Royalties — pagamento", NatureRemessa},

	// Capital (capital flows)
	{"30001", "Investimento estrangeiro direto — ingresso", NatureIngresso},
	{"30002", "Retorno de IED — remessa", NatureRemessa},
	{"30010", "Empréstimo externo — ingresso", NatureIngresso},
	{"30011", "Empréstimo externo — pagamento de principal", NatureRemessa},
	{"30012", "Empréstimo externo — pagamento de juros", NatureRemessa},

	// Transferências unilaterais (transfers)
	{"40001", "Manutenção de residentes — remessa", NatureRemessa},
	{"40002", "Manutenção de residentes — ingresso", NatureIngresso},

	// Turismo + cartão (travel + cards)
	{"50001", "Viagens internacionais — turismo", NatureRemessa},
	{"50002", "Cartão de crédito internacional", NatureRemessa},

	// Outros
	{"60001", "Conversão entre moedas estrangeiras", NatureConversao},
	{"63010", "Operação financeira — derivativo", NatureRemessa},
	{"99999", "Outros — código residual", NatureRemessa},
}

// Classifier resolves nature codes by code or by free-text hint.
type Classifier struct {
	byCode map[string]NatureCode
	// keyword index — first-match wins (order matters).
	keywords []keywordRule
}

type keywordRule struct {
	keyword string
	code    string
}

// NewClassifier constructs a Classifier seeded with the builtin catalog plus
// optional `extra` rows (from refdata).
//
// Keyword rules are ordered by specificity (most specific first). The first
// rule whose `keyword` is a case-insensitive substring of the hint wins; if
// none match, Classify returns ErrUnknown. Rules target codes from the
// generated catalogue (`AllNatureCodes`) so the slim NatureCode returned by
// Classify always carries the canonical description + direction.
func NewClassifier(extra ...NatureCode) *Classifier {
	c := &Classifier{
		byCode:   make(map[string]NatureCode, len(builtin)+len(extra)),
		keywords: defaultKeywordRules(),
	}
	for _, n := range builtin {
		c.byCode[n.Code] = n
	}
	for _, n := range extra {
		c.byCode[n.Code] = n
	}
	return c
}

// defaultKeywordRules is the boot keyword index. Rules ordered most specific →
// most general; first match wins. Designed against
// data/bacen/golden-classifications.csv at ≥95% hit rate.
func defaultKeywordRules() []keywordRule {
	return []keywordRule{
		// ── DERIVATIVOS (most specific phrases first) ──────────────────────
		{"ndf", "70000"},
		{"non-deliverable forward", "70000"},
		{"non deliverable forward", "70000"},
		{"swap", "70002"},
		{"forward", "70001"},

		// ── VASP / virtual assets (direction-specific rules FIRST) ─────────
		{"recebimento de pagamento em ativo virtual", "90000"},
		{"recebimento em criptomoeda", "90000"},
		{"recebimento em cripto", "90000"},
		{"venda de criptoativo a comprador", "90000"},
		{"venda de cripto", "90000"},
		{"recebimento por ativos virtuais", "90000"},
		{"compra de criptoativo", "90001"},
		{"compra de cripto", "90001"},
		{"stablecoin", "90001"},
		{"wallet", "90001"},
		{"ativo virtual", "90001"},
		{"criptomoeda", "90000"},
		{"criptoativo", "90001"},
		{"cripto", "90001"},

		// ── TURISMO (must precede CARTAO + spot-cash rules) ─────────────────
		{"viagem internacional em espécie", "40001"},
		{"moeda para viagem", "40001"},
		{"dólar em espécie", "40001"},
		{"recebimentos de turistas", "40002"},
		{"hospedagem de turista", "40002"},
		{"turista estrangeiro", "40002"},
		{"pacote turístico", "40002"},
		{"viagem", "40000"},
		{"turismo", "40000"},

		// ── CARTAO ─────────────────────────────────────────────────────────
		{"cartão de débito", "50001"},
		{"cartão de crédito", "50000"},
		{"credit card", "50000"},
		{"debit card", "50001"},

		// ── TRANSFERENCIAS (life events) ───────────────────────────────────
		{"tratamento médico", "30300"},
		{"hospitalar", "30300"},
		{"médico-hospitalar", "30300"},
		{"mensalidade escolar", "30200"},
		{"tuition", "30200"},
		{"estudos", "30200"},
		{"despesas com estudo", "30200"},
		{"aposentadoria", "30003"},
		{"pensão", "30003"},
		{"previdenciário", "30003"},
		{"salário", "30002"},
		{"expatriado", "30002"},
		{"remuneração", "30002"},
		{"herança", "30001"},
		{"espólio", "30001"},
		{"legado", "30000"},
		{"doação", "30000"},
		{"donativo", "30000"},
		{"manutenção de residentes", "30100"},
		{"manutenção de familiar", "30100"},
		{"sustento de familiar", "30100"},
		{"mesada", "30100"},
		{"familiar no exterior", "30100"},
		{"familiar residente no exterior", "30100"},

		// ── RENDA — royalty + interest ─────────────────────────────────────
		{"juros sobre capital próprio", "20002"},
		{"jcp", "20002"},
		{"obra literária", "60100"},
		{"royalties de obra literária", "60100"},
		{"direitos autorais", "60100"},
		{"direito autoral", "60100"},
		{"royalty de marca", "60201"},
		{"royalty de patente", "60200"},
		{"royalties marca", "60201"},
		{"royalties tecnologia", "60200"},
		{"licença de uso de tecnologia", "60200"},
		{"tecnologia patenteada", "60200"},
		{"royalty", "60200"},
		{"royalties", "60200"},
		{"franquia internacional", "60201"},
		{"licença de marca", "60201"},
		{"juros de depósito", "60000"},
		{"juros remuneratórios", "60000"},
		{"rendimento de aplicação", "60000"},
		{"rendimento de aplicação financeira", "60000"},

		// ── CAPITAL — loans + investment ───────────────────────────────────
		{"juros de empréstimo", "21002"},
		{"juros remetidos", "21002"},
		{"juros sobre dívida", "21002"},
		{"amortização de empréstimo", "21001"},
		{"pagamento de principal de empréstimo", "21001"},
		{"quitação parcial de dívida", "21001"},
		{"empréstimo intercompany", "21100"},
		{"mútuo com empresa", "21100"},
		{"intragrupo", "21100"},
		{"financiamento de importação", "21200"},
		{"trade finance", "21200"},
		{"linha de crédito para importação", "21200"},
		{"empréstimo externo", "21000"},
		{"financiamento internacional", "21000"},
		{"bond internacional", "21000"},
		// Portfolio: direction-specific rules MUST precede the generic ones.
		{"retorno de investimento em portfólio", "20101"},
		{"resgate de aplicação", "20101"},
		{"saída de recursos de investimento", "20101"},
		{"aplicação financeira de investidor estrangeiro", "20100"},
		{"ações b3", "20100"},
		{"portfólio", "20100"},
		{"investidor não-residente", "20100"},
		{"títulos públicos", "20100"},
		// Dividendos must precede the generic IED rules — "investidor" appears in both.
		{"dividendos", "20002"},
		{"dividend", "20002"},
		{"desinvestimento", "20001"},
		{"repatriação de capital", "20001"},
		{"retorno de investimento", "20001"},
		{"ied", "20000"},
		{"investimento estrangeiro direto", "20000"},
		{"aporte de capital", "20000"},
		{"foreign direct investment", "20000"},
		// CBE direction-specific rules (resgate/retorno = inbound) MUST precede
		// the generic offshore/CBE rules to avoid the inbound→outbound flip.
		{"resgate de fundo offshore", "22001"},
		{"retorno de aplicação no exterior", "22001"},
		{"cbe", "22001"},
		{"repatriação", "22001"},
		{"fundo no exterior", "22000"},
		{"capitais brasileiros no exterior", "22000"},
		{"offshore", "22000"},

		// ── SERVICOS ───────────────────────────────────────────────────────
		{"indenização de sinistro", "15201"},
		{"indenização de seguro", "15201"},
		{"pagamento de sinistro", "15201"},
		{"prêmio de seguro", "15200"},
		{"prêmios de seguro", "15200"},
		{"seguro de carga", "15200"},
		{"seguro de transporte", "15200"},
		{"insurance premium", "15200"},
		{"frete", "15100"},
		{"transporte marítimo", "15100"},
		{"transporte internacional", "15100"},
		{"shipping", "15100"},
		{"telecomunicação", "15300"},
		{"telecomunicações", "15300"},
		{"telecommunications", "15300"},
		{"roaming", "15300"},
		{"construção", "15400"},
		{"engenharia civil", "15400"},
		{"consultoria", "15001"},
		{"serviços técnicos", "15001"},
		{"honorários", "15001"},
		{"governamentais", "15500"},
		{"consular", "15500"},
		{"embaixada", "15500"},
		{"serviços diversos", "15000"},
		{"serviços gerais", "15000"},

		// ── COMERCIAL (must come AFTER service rules) ──────────────────────
		{"acc adiantamento", "10101"},
		{"recebimento antecipado", "10101"},
		{"adiantamento de cliente", "10101"},
		{"adiantamento contrato câmbio exportação", "10101"},
		{"pré-pagamento de importação", "10201"},
		{"pagamento antecipado para importação", "10201"},
		{"pagamento antecipado ao fornecedor", "10201"},
		{"pagamento adiantado para importação", "10201"},
		{"exportação", "10000"},
		{"vendas externas", "10000"},
		{"export", "10000"},
		{"importação", "10200"},
		{"import", "10200"},
		{"compra de máquinas", "10200"},
		{"bens de capital", "10200"},

		// ── OUTROS — fallback bucket (LAST) ────────────────────────────────
		{"não classificada", "80000"},
		{"não especificada", "80000"},
		{"conversão entre moedas", "80000"},
		{"demais operações", "80000"},
	}
}

// ByCode returns the NatureCode for an exact code.
//
// Resolution order:
//  1. Generated catalogue (AllNatureCodes — full 46-code BACEN Circ 3.690 set),
//     compressed to the slim NatureCode shape via deriveNature().
//  2. Constructor-supplied extra rows + builtin seed (legacy path).
//
// Returns false on miss.
func (c *Classifier) ByCode(code string) (NatureCode, bool) {
	code = strings.TrimSpace(code)
	if full, ok := AllNatureCodes[code]; ok && full.Active {
		return NatureCode{
			Code:        full.Code,
			Description: full.DescriptionPT,
			Nature:      deriveNature(full.Direction),
		}, true
	}
	n, ok := c.byCode[code]
	return n, ok
}

// deriveNature maps the generated catalogue's Direction (INGRESSO/REMESSA/BIDIRECTIONAL)
// to the slim Nature type. BIDIRECTIONAL collapses to CONVERSAO (commercial intent
// matches the swap/conversion class of operations in legacy callers).
func deriveNature(direction string) Nature {
	switch direction {
	case "INGRESSO":
		return NatureIngresso
	case "REMESSA":
		return NatureRemessa
	case "BIDIRECTIONAL":
		return NatureConversao
	default:
		return NatureRemessa
	}
}

// Classify resolves a nature from a free-text hint (case-insensitive substring match).
// Returns ErrUnknown if no rule matches.
//
// Resolution order:
//  1. First matching keyword rule selects a code (most-specific rules go first
//     in the slice).
//  2. The code is resolved via ByCode, which consults the generated
//     AllNatureCodes catalogue then falls back to the legacy builtin map. If
//     neither has the code (rule misconfigured), returns ErrUnknown.
func (c *Classifier) Classify(hint string) (NatureCode, error) {
	h := strings.ToLower(strings.TrimSpace(hint))
	if h == "" {
		return NatureCode{}, fmt.Errorf("%w: hint is empty", ErrUnknown)
	}
	for _, rule := range c.keywords {
		if strings.Contains(h, rule.keyword) {
			n, ok := c.ByCode(rule.code)
			if !ok {
				return NatureCode{}, fmt.Errorf("%w: rule %q targets unknown code %q", ErrUnknown, rule.keyword, rule.code)
			}
			return n, nil
		}
	}
	return NatureCode{}, fmt.Errorf("%w: no rule for %q", ErrUnknown, hint)
}

// All returns the seeded set (in catalog order).
func (c *Classifier) All() []NatureCode {
	out := make([]NatureCode, 0, len(c.byCode))
	for _, n := range builtin {
		out = append(out, n)
	}
	return out
}
