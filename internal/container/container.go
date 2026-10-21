// Package container wires application services + their repositories so
// cmd/api/main.go has a single place to depend on.
//
// Backend selection: EXCHANGEOS_REPO_BACKEND=memory|postgres (default memory).
//
//	memory   — in-memory bootstrap repos (modules/<bc>/infrastructure/memory).
//	postgres — pgx/v5 repos (modules/<bc>/infrastructure/postgres) backed by the
//	           shared CRDB hub TLS pool.
package container

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	"github.com/revenutech/exchangeos/internal/config"
	"github.com/revenutech/exchangeos/internal/db"
	"github.com/revenutech/exchangeos/internal/eventbus"

	quoteapp "github.com/revenutech/exchangeos/modules/quote/application"
	qdomain "github.com/revenutech/exchangeos/modules/quote/domain"
	quotemem "github.com/revenutech/exchangeos/modules/quote/infrastructure/memory"
	quotepg "github.com/revenutech/exchangeos/modules/quote/infrastructure/postgres"

	refapp "github.com/revenutech/exchangeos/modules/refdata/application"
	refdomain "github.com/revenutech/exchangeos/modules/refdata/domain"
	refmem "github.com/revenutech/exchangeos/modules/refdata/infrastructure/memory"
	refpg "github.com/revenutech/exchangeos/modules/refdata/infrastructure/postgres"
	refpricing "github.com/revenutech/exchangeos/modules/refdata/infrastructure/pricing"

	tradeapp "github.com/revenutech/exchangeos/modules/trade/application"
	trademem "github.com/revenutech/exchangeos/modules/trade/infrastructure/memory"
	tradepg "github.com/revenutech/exchangeos/modules/trade/infrastructure/postgres"

	clsapp "github.com/revenutech/exchangeos/modules/cls_settlement/application"
	clsmem "github.com/revenutech/exchangeos/modules/cls_settlement/infrastructure/memory"

	netapp "github.com/revenutech/exchangeos/modules/netreport/application"
	netmem "github.com/revenutech/exchangeos/modules/netreport/infrastructure/memory"
	netpg "github.com/revenutech/exchangeos/modules/netreport/infrastructure/postgres"

	payapp "github.com/revenutech/exchangeos/modules/payin/application"
	paymem "github.com/revenutech/exchangeos/modules/payin/infrastructure/memory"
	paypg "github.com/revenutech/exchangeos/modules/payin/infrastructure/postgres"

	posapp "github.com/revenutech/exchangeos/modules/position/application"
	posmem "github.com/revenutech/exchangeos/modules/position/infrastructure/memory"
	pospg "github.com/revenutech/exchangeos/modules/position/infrastructure/postgres"

	riskapp "github.com/revenutech/exchangeos/modules/risk/application"
	riskmem "github.com/revenutech/exchangeos/modules/risk/infrastructure/memory"
	riskpg "github.com/revenutech/exchangeos/modules/risk/infrastructure/postgres"

	clspg "github.com/revenutech/exchangeos/modules/cls_settlement/infrastructure/postgres"

	adminapp "github.com/revenutech/exchangeos/modules/admin/application"
	adminmem "github.com/revenutech/exchangeos/modules/admin/infrastructure/memory"
	adminpg "github.com/revenutech/exchangeos/modules/admin/infrastructure/postgres"

	complapp "github.com/revenutech/exchangeos/modules/compliance/application"
	complmem "github.com/revenutech/exchangeos/modules/compliance/infrastructure/memory"
	complpg "github.com/revenutech/exchangeos/modules/compliance/infrastructure/postgres"

	cfcapapp "github.com/revenutech/exchangeos/modules/cfets_capture/application"
	cfcapmem "github.com/revenutech/exchangeos/modules/cfets_capture/infrastructure/memory"
	cfcappg "github.com/revenutech/exchangeos/modules/cfets_capture/infrastructure/postgres"

	cfconapp "github.com/revenutech/exchangeos/modules/cfets_confirmation/application"
	cfconmem "github.com/revenutech/exchangeos/modules/cfets_confirmation/infrastructure/memory"
	cfconpg "github.com/revenutech/exchangeos/modules/cfets_confirmation/infrastructure/postgres"

	"github.com/revenutech/exchangeos/pkg/bacen"
)

// Container holds all application services + their dependencies.
type Container struct {
	Config *config.Config
	Pool   *pgxpool.Pool // non-nil when Backend == "postgres"

	RefData           *refapp.Service
	Quote             *quoteapp.Service
	Trade             *tradeapp.Service
	Settlement        *clsapp.Service
	PayIn             *payapp.Service
	NetReport         *netapp.Service
	Risk              *riskapp.Service
	Position          *posapp.Service
	Compliance        *complapp.Service
	Admin             *adminapp.Service
	CFETSCapture      *cfcapapp.Service
	CFETSConfirmation *cfconapp.Service
	Pricing           quoteapp.PricingEngine
	SpotBook          *refdomain.SpotRateBook // exposed so worker feeders can publish rates
	EventBus          *eventbus.Bus           // in-process pub/sub (temporary; Kafka outbox in MS-023g)

	// Memory-only handles kept for tests/bootstrap convenience (nil if backend != memory).
	MemRefDataCurrencies *refmem.CurrencyRepo
	MemRefDataCalendars  *refmem.CalendarRepo
	MemRefDataBICs       *refmem.BICRepo
	MemRefDataSSIs       *refmem.SSIRepo
	MemQuoteQuotes       *quotemem.QuoteRepo
	MemQuoteRFQs         *quotemem.RFQRepo
	MemQuotePublisher    *quotemem.NoopPublisher
	MemTradeRepo         *trademem.TradeRepo
	MemTradePublisher    *trademem.NoopPublisher
}

// New constructs a Container honouring cfg.Repos.Backend.
// For backend=postgres, a pgxpool is built; the caller must invoke Close() to release it.
//
// Pricing wiring:
//   - A live SpotRateBook (5s freshness) is created in every container.
//   - Engine = SpotRateBook + flat half-spread (0.0002 default).
//   - A development seed populates EUR/USD = 1.0800 so the smoke endpoint /v1/quotes works
//     without an external feeder. Production wiring replaces the seed with a market-data
//     consumer pushing into SpotBook.
func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	c := &Container{Config: cfg, EventBus: eventbus.New()}

	// Live spot-rate book + spread policy → real PricingEngine.
	c.SpotBook = refdomain.NewSpotRateBook(5 * time.Second)
	c.Pricing = refpricing.New(c.SpotBook, refpricing.FlatSpreadPolicy{
		Value: decimal.RequireFromString("0.0002"),
	})
	seedSpotBookDev(c.SpotBook)

	switch cfg.Repos.Backend {
	case "memory":
		c.wireMemory()
	case "postgres":
		pool, err := db.New(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("container: db pool: %w", err)
		}
		c.Pool = pool
		c.wirePostgres(pool)
	default:
		return nil, fmt.Errorf("container: unsupported backend %q", cfg.Repos.Backend)
	}

	c.wireEventHandlers()
	return c, nil
}

// wireEventHandlers subscribes cross-context reactors to the in-process bus.
// Currently: quote.accepted.v1 → trade.BookTrade (via QuoteAcceptedHandler).
func (c *Container) wireEventHandlers() {
	if c.Quote == nil || c.Trade == nil {
		return
	}
	handler := &tradeapp.QuoteAcceptedHandler{
		Trades: c.Trade,
		QuoteLookup: func(ctx context.Context, quoteID uuid.UUID) (tradeapp.AcceptedQuoteView, error) {
			// Look up via memory repo when available; postgres path returns ErrUnimplemented
			// until a quote.application read-model accessor is added.
			if c.MemQuoteQuotes == nil {
				return tradeapp.AcceptedQuoteView{}, fmt.Errorf("quote lookup unavailable in %s backend", c.Config.Repos.Backend)
			}
			q, err := c.MemQuoteQuotes.Get(ctx, quoteID)
			if err != nil {
				return tradeapp.AcceptedQuoteView{}, err
			}
			// Every field below comes off the aggregate. This adapter used to
			// substitute literals for the three the Quote did not carry —
			// BuyerBIC "DEUTDEFF", SellerBIC "CHASUS33", Venue "CLS" — and take
			// q.Mid() as the deal rate because there was no side to choose
			// bid-vs-ask with. The Quote now carries counterparties, side and
			// venue, so nothing is inferred here.
			//
			// Buyer/seller and the deal rate are derived by the aggregate rather
			// than assembled here: which party buys the base and whether the
			// trade prints on the bid or the ask are both consequences of the
			// side, and that rule belongs in the domain.
			return tradeapp.AcceptedQuoteView{
				TenantID:    q.TenantID(),
				BuyerBIC:    q.BuyerBIC(),
				SellerBIC:   q.SellerBIC(),
				BaseCCY:     q.BaseCCY(),
				QuoteCCY:    q.QuoteCCY(),
				NotionalCCY: q.NotionalCCY(),
				Notional:    q.Notional(),
				DealRate:    q.DealRate(),
				Venue:       q.Venue(),
				AcceptedAt:  time.Now().UTC(),
			}, nil
		},
	}
	c.EventBus.Subscribe("quote.accepted.v1", func(ctx context.Context, e eventbus.Event) error {
		de, ok := e.(qdomain.DomainEvent)
		if !ok {
			return nil
		}
		return handler.Handle(ctx, de)
	})
}

// seedSpotBookDev populates a handful of pairs so dev / smoke tests work without
// a live market-data feeder. Production feeders replace this entirely.
func seedSpotBookDev(book *refdomain.SpotRateBook) {
	now := time.Now().UTC()
	for _, r := range []refdomain.SpotRate{
		{BaseCCY: "EUR", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.0800"), AsOf: now},
		{BaseCCY: "GBP", QuoteCCY: "USD", Mid: decimal.RequireFromString("1.2700"), AsOf: now},
		{BaseCCY: "USD", QuoteCCY: "JPY", Mid: decimal.RequireFromString("145.00"), AsOf: now},
		{BaseCCY: "USD", QuoteCCY: "BRL", Mid: decimal.RequireFromString("5.10"), AsOf: now},
		{BaseCCY: "USD", QuoteCCY: "CAD", Mid: decimal.RequireFromString("1.36"), AsOf: now},
	} {
		_ = book.Put(r)
	}
}

// Close releases resources held by the container.
func (c *Container) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
}

func (c *Container) wireMemory() {
	currencies := refmem.NewCurrencyRepo()
	calendars := refmem.NewCalendarRepo()
	bics := refmem.NewBICRepo()
	ssis := refmem.NewSSIRepo()
	c.MemRefDataCurrencies = currencies
	c.MemRefDataCalendars = calendars
	c.MemRefDataBICs = bics
	c.MemRefDataSSIs = ssis
	c.RefData = refapp.NewService(currencies, calendars, bics, ssis)

	quotes := quotemem.NewQuoteRepo()
	rfqs := quotemem.NewRFQRepo()
	c.MemQuoteQuotes = quotes
	c.MemQuoteRFQs = rfqs
	c.MemQuotePublisher = quotemem.NewNoopPublisher() // retained as a tap for tests
	c.Quote = quoteapp.NewService(quotes, rfqs, c.Pricing,
		eventbus.QuotePublisher{Bus: c.EventBus},
		quoteapp.Options{DefaultQuoteTTL: 10 * time.Second},
	)

	tradeRepo := trademem.NewTradeRepo()
	c.MemTradeRepo = tradeRepo
	c.MemTradePublisher = trademem.NewNoopPublisher()
	c.Trade = tradeapp.NewService(tradeRepo, eventbus.TradePublisher{Bus: c.EventBus})

	c.wireSettlement()
}

// wireSettlement constructs the cls_settlement + payin + netreport services
// against in-memory repositories. Postgres-backed repos are a follow-up.
func (c *Container) wireSettlement() {
	cycleRepo := clsmem.NewCycleRepo()
	clsPub := clsmem.NewNoopPublisher()
	c.Settlement = clsapp.NewService(cycleRepo, clsPub)

	payRepo := paymem.NewRepo()
	payPub := paymem.NewNoopPublisher()
	c.PayIn = payapp.NewService(payRepo, payPub)

	netRepo := netmem.NewRepo()
	c.NetReport = netapp.NewService(netRepo)

	// Risk + Position — both in-memory; migration 000007 lands the postgres impl.
	c.Risk = riskapp.NewService(riskmem.NewRepo())
	c.Position = posapp.NewService(posmem.NewRepo())

	c.wireComplianceAdmin()
}

// wireComplianceAdmin constructs compliance + admin services. Uses pkg/bacen
// Classifier + IOFCalculator with default catalogs.
func (c *Container) wireComplianceAdmin() {
	c.Compliance = complapp.NewService(
		bacen.NewClassifier(),
		bacen.NewIOFCalculator(),
		complmem.NewClassificationRepo(),
		complmem.NewIOFRepo(),
		complmem.NewReportRepo(),
		complmem.NewScreeningRepo(),
		// Fail-closed até que um provedor real de listas (OFAC/UN/EU/COAF)
		// seja ligado: screening errors out em vez de liberar a contraparte.
		complapp.UnavailableScreener{},
	)
	c.Admin = adminapp.NewService(adminmem.NewEventRepo(), adminmem.NewEODJobRepo())

	c.CFETSCapture = cfcapapp.NewService(cfcapmem.NewRepo(), cfcapmem.NewNoopPublisher())
	c.CFETSConfirmation = cfconapp.NewService(cfconmem.NewRepo(), cfconmem.NewNoopPublisher())
}

func (c *Container) wirePostgres(pool *pgxpool.Pool) {
	currencies := refpg.NewCurrencyRepo(pool)
	calendars := refpg.NewCalendarRepo(pool)
	bics := refpg.NewBICRepo(pool)
	ssis := refpg.NewSSIRepo(pool)
	c.RefData = refapp.NewService(currencies, calendars, bics, ssis)

	quotes := quotepg.NewQuoteRepo(pool)
	rfqs := quotepg.NewRFQRepo(pool)
	c.MemQuotePublisher = quotemem.NewNoopPublisher() // retained as a tap for tests
	c.Quote = quoteapp.NewService(quotes, rfqs, c.Pricing,
		eventbus.QuotePublisher{Bus: c.EventBus},
		quoteapp.Options{DefaultQuoteTTL: 10 * time.Second},
	)

	tradeRepo := tradepg.NewTradeRepo(pool)
	c.MemTradePublisher = trademem.NewNoopPublisher()
	c.Trade = tradeapp.NewService(tradeRepo, eventbus.TradePublisher{Bus: c.EventBus})

	c.wireSettlementPostgres(pool)
}

// wireSettlementPostgres uses postgres-backed repos for cls_settlement + payin +
// netreport + risk + position. Publishers stay on memory-noop until Kafka outbox
// lands (separate workstream).
func (c *Container) wireSettlementPostgres(pool *pgxpool.Pool) {
	cycleRepo := clspg.NewCycleRepo(pool)
	clsPub := clsmem.NewNoopPublisher()
	c.Settlement = clsapp.NewService(cycleRepo, clsPub)

	payRepo := paypg.NewPayInRepo(pool)
	payPub := paymem.NewNoopPublisher()
	c.PayIn = payapp.NewService(payRepo, payPub)

	netRepo := netpg.NewNetReportRepo(pool)
	c.NetReport = netapp.NewService(netRepo)

	c.Risk = riskapp.NewService(riskpg.NewLimitRepo(pool))
	c.Position = posapp.NewService(pospg.NewPositionRepo(pool))

	c.wireComplianceAdminPostgres(pool)
}

// wireComplianceAdminPostgres mirrors wireComplianceAdmin but with postgres-backed
// repos for compliance + admin + cfets (both BCs).
func (c *Container) wireComplianceAdminPostgres(pool *pgxpool.Pool) {
	c.Compliance = complapp.NewService(
		bacen.NewClassifier(),
		bacen.NewIOFCalculator(),
		complpg.NewClassificationRepo(pool),
		complpg.NewIOFRepo(pool),
		complpg.NewReportRepo(pool),
		complpg.NewScreeningRepo(pool),
		// Fail-closed até que um provedor real de listas (OFAC/UN/EU/COAF)
		// seja ligado: screening errors out em vez de liberar a contraparte.
		complapp.UnavailableScreener{},
	)
	c.Admin = adminapp.NewService(adminpg.NewEventRepo(pool), adminpg.NewEODJobRepo(pool))

	c.CFETSCapture = cfcapapp.NewService(cfcappg.NewCFETSCaptureRepo(pool), cfcapmem.NewNoopPublisher())
	c.CFETSConfirmation = cfconapp.NewService(cfconpg.NewCFETSConfirmationRepo(pool), cfconmem.NewNoopPublisher())
}

// stubPricing was removed in 4.9.0 — Container now uses refdata/infrastructure/pricing.Engine
// against a live SpotRateBook. See seedSpotBookDev for the development seed.
