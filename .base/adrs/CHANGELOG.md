# Catálogo de ADRs — Changelog

## v1.1.0 (2026-10-23)

### Added

- **ADR-009** `RADR.024.009.01` — ExchangeOS passa a ter dois produtos
  coexistentes: interbancário (atual) e câmbio de varejo (novo, originado
  pelo app). Conteúdo decidido pelo dono em 2026-10-23 (contraparte: Revenu
  como principal via mesa `treasuryos` · enquadramento: eFX Res. BCB 561 ·
  pares M1: USD/BRL · limites: USD 10k/op, 10k/dia, 50k/mês · 5 eventos
  canônicos · 4 rotas em `/mobile/v1/exchangeos`). Status: `DRAFT`
  aguardando revisão final do documento. Pré-requisito de `MS-163 2.3`
  (`RTSK.080.823.01`) do programa `integracao-mobile-tempo-real`.

### Changed

- **ADR-009** refinada com três adições factuais (sem mudar as decisões
  do dono): (a) seção «Estado medido no app iOS» registra o teto local de
  R$ 1 milhão em `ExchangeCalculator.limitTotalCents` que vai sair; (b)
  Decisão § 4 passa a citar a proibição de fracionamento (Res. BCB 277
  arts. 29, 49) com teste de aceite `TestRetailFX_SplitOperationRefused`
  — origem MS-201 3.5; (c) *forward reference* ao plano
  `limites-de-operacao` (MS-200..202) como futuro gestor único de
  limites no RiskOS, com `RN_FX_015` NOP permanecendo no exchangeos.

## v1.0.0 (2026-10-22)

### Added

- Catálogo criado no padrão da plataforma (`baremetal/.base/adrs/`,
  `treasuryos/.base/adrs/`), faixa `220`–`249` do `024` (decisão do dono,
  `baremetal/.base/plans/CONVENTIONS.md` § 3.3.5).
- `CONVENTIONS.md`, `TEMPLATE.md`, `index.md` com o critério de ADR
  (`onboardos-revenu/.base/adrs/README.md`: alternativa real · porta cara
  de reabrir · atravessa contextos · porquê não dedutível do código).
- Levantamento de **8 candidatos a ADR** no `index.md`, lavrados a partir
  do `CLAUDE.md`, do `.base/plans/00-governance/index.md` e dos últimos
  trinta commits.
