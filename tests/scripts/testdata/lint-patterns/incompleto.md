# Fixture — incomplete pattern (no Anti-pattern, no code, Related only cites itself)

## FX-GP-902 — Decimal precision

**Context:** Money arithmetic.

**Problem:** float64 loses precision.

**Solution:** Use decimal.Decimal.

**Example:** `pkg/decimal`.

**Related:** FX-GP-902.
