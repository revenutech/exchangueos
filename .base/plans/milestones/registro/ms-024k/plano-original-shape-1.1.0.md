# MS-024k — Pattern Catalogue Build-out (850 patterns)

| Field | Value |
|-------|-------|
| **Code** | MS-024k |
| **Name** | pattern-catalogue-buildout |
| **Phase** | F-OPS-PROD |
| **Sprint** | Background — concurrent with sprints 1-4 |
| **Status** | BACKLOG |
| **Owner** | Distributed — one pattern per code review |
| **Dependencies** | None |

## Why this milestone

`.base/plans/01-architecture/patterns/` indexes **20 catalogs totalling 850 planned patterns**. Currently **27 patterns** have full Context/Problem/Solution/Example/Anti-pattern/Related blocks; the remaining 823 are named placeholders. A pattern catalogue with > 95% placeholders is documentation theatre — useful only when filled.

## Description

Convert the 823 placeholder patterns into real entries over the MS-024 cycle, using a "pattern-per-PR" tax: any code PR touching a documented pattern area must either fill the pattern or open a tracked debt entry. Goal is **300 patterns fully written by end of cycle** (out of 850), prioritising the most frequently-cited ones (FX-GP-*, FX-DDD-*, FX-EDA-*, FX-CP-*, FX-COMMIT-*).

## Acceptance Criteria

- [ ] Pattern template formalised in `.base/plans/01-architecture/patterns/TEMPLATE.md` (6 sections, ≤ 400 words each)
- [ ] Lint script `scripts/lint-patterns.sh` validating each pattern file has all 6 sections + ≥ 1 code example + ≥ 1 anti-pattern + ≥ 1 cross-link
- [ ] Lefthook pre-commit hook running pattern lint when files under `patterns/` change
- [ ] **300 patterns** fully written by milestone close (target distribution: 60 FX-GP, 50 FX-DDD, 50 FX-EDA, 40 FX-CP, 30 FX-COMMIT, 30 FX-QA, 40 misc)
- [ ] Each filled pattern cross-references at least one production code path (file:line)
- [ ] Patterns README updated weekly with progress bar
- [ ] Quarterly pattern-review session (1h) to retire deprecated patterns + promote useful new ones
- [ ] Top 10 most-cited patterns measured via `grep -r "FX-.*-NNN" modules/ pkg/ | sort | uniq -c | sort -rn`

## Deliverables

- `.base/plans/01-architecture/patterns/TEMPLATE.md`
- `scripts/lint-patterns.sh`
- 273 new fully-filled pattern files (27 existing + 273 new = 300)
- Updated patterns `README.md` with progress + top-10
- Pattern-review meeting notes per quarter

## Cross-References

- `.base/plans/01-architecture/patterns/README.md` — current index
- `CLAUDE.md` workflow §2 (TDD-first) consumed by patterns
- ISO 27001 control 5.37 (documented operating procedures)
