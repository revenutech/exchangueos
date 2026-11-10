---
data: 2026-11-10
tarefa: 1.1
estado: ☑️
evidencia: migração 000010 (`archive_policy` com 24 políticas e `archive_manifest`) e o teste de ida e volta das migrações; revisada pela IdentityOS; N1 verde, na develop em 64e1d03; N2 do lote 9.2 pendente
pr:
---
Tarefa `RTSK.024.1001.01` (MS-024b 1.1), 2026-11-10, lote 9.2 da etapa 9 (código liberado pelo dono por select na Core em 2026-11-10).

- **entrega:** `migrations/000010_create_archive_tables.{up,down}.sql`; `tests/ci-local/migracoes-ida-e-volta.sh` (CockroachDB TLS descartável, verify-full; up, checks, idempotência, down com os checks falhando, up de novo) e o passo `migracoes` no `passos.conf` (pista com banco).
- **24 e não 25:** o 25 vinha do plano 1.1.0 (`--max-tables 25`, que é teto e comporta 24). O README tem 29 tabelas; 5 dizem «Never» (currencies, calendars, currency_pairs, netting_cutoffs, bic_records) e ficam sem linha, porque `age_threshold` é NOT NULL. Decisão do dono via Core em 2026-11-10; aceite ajustado.
- **revisão:** IdentityOS aprovou `2d1a946`; concorda com as duas tabelas sem `tenant_id` (configuração da plataforma e auditoria do arquivador).
- **achados para a 1.4, a 1.6 e o arquivador:** o README diz «anonimizar» onde a política faz hard-delete (RULE-008; decide a 1.6 com o DPO); a cópia no GCS precisa de lifecycle com a mesma retenção (linha ✏️ 1.14); o manifesto não diz o tenant do lote (lote por tabela e tenant, ou coluna `tenant_ids`); `screening_results` ligados a COS têm piso de 5 anos (Lei 9.613, art. 10).
- **N1:** pista com banco: «ida e volta verde (12 migrações, 6 checks)»; o vermelho do TDD (`64d8fe2`, sem a migração) falhou.
- **veredito do lote:** compara com a linha de base do accountos (p75 2,7 h). A do exchangeos ficou degenerada (`--medir-linha-de-base --desde 2026-11-01 --ate 2026-11-09`: 19 merges, 18 de registro; mediana 0,0 h, p75 0,0 h, p90 1,1 h). Decisão do dono por select na Core em 2026-11-10.

```prova
n1: 2d1a946197927ef43fe84446d9bf41507e7a8957 rodada exchangeos-migracoes@2026-11-10T18:49:08Z
n2: pendente — N2 única do lote 9.2, depois do último merge
integracao: develop 64e1d031305678d6a0885c39b09925ffe20f0480
revisor: sessão IdentityOS — aprovado em 2d1a946 em 2026-11-10 (PII: retenção LGPD)
```
