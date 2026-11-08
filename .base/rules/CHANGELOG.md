# Catálogo de Regras — Changelog

## v1.0.0 (2026-10-22)

### Added

- Catálogo criado no padrão da plataforma (`baremetal/.base/rules/`,
  `treasuryos/.base/rules/`), faixa `220`–`249` do `024` (decisão do dono,
  `baremetal/.base/plans/CONVENTIONS.md` § 3.3.5).
- **11 regras** com o alias legado `RN_FX_*` migradas, com status medido
  pelo caminho de produção: **5 `ENFORCED`, 1 `AUDIT`, 5 `PROPOSED`**.
- Áreas estruturadas em 10 arquivos (`gov`, `trade`, `quote`, `settle`,
  `cfets`, `sett2`, `refdata`, `risk`, `comp`, `eod`).
- Achado registrado em `index.md`: o `CLAUDE.md` fala em «50 regras
  `RN_FX_001..050`»; a medição no código mostra 11 aliases, dos quais 5
  no caminho de produção. As 39 restantes são intenção de catálogo
  (declaradas em SHACL *shapes* de `.base/aasc/ontology/compliance/`),
  não controle.
