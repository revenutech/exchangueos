# MS-024c — Credential Rotator (loop real)

**Code:** `RMLS.024.011.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=11; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Security + Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024c/plano-original-shape-1.1.0.md](../registro/ms-024c/plano-original-shape-1.1.0.md)

## Por quê

`cmd/cred-rotator/` tem esqueleto + catálogo de 14 secrets documentado em `docs/integrations/identityos.md`. O loop mensal real contra Vault (gerar `client_secret`, empurrar ao Vault, empurrar à IdentityOS/KeycloakOS, sinalizar pods para recarga via SIGHUP ou rolling restart, alertar em falha) não está implementado.

## Descrição

Construir o rotator real dirigido por cron que orquestra rotação mensal de `client_secret` OAuth2 para os 14 clientes M2M usados por ExchangeOS + integrações irmãs. Deve ser idempotente, observável e capaz de operação sem assistência sob identidade WIF.

## Fase 1 — Implementação

### Etapa 1.1 — Catálogo + orquestrador

- **1.1** `RTSK.024.1101.01` ✏️ Criar `config/secrets-catalog.yaml` listando 14 clientes M2M + paths Vault + backends IdP. *Aceite:* YAML valida contra schema, 14 entradas. *Entrega:* `config/secrets-catalog.yaml`. *Escopo:* `config/`. *Espera:* dono — definição final dos 14 clientes. [registro](../registro/README.md)
- **1.2** `RTSK.024.1102.01` ✏️ Lock distribuído via Vault `kv/locks/cred-rotator/<secret>`. *Aceite:* lock impede rotação concorrente do mesmo secret, libera em sucesso ou falha. *Entrega:* `internal/credrotator/vault.go` (seção lock). *Escopo:* `internal/credrotator/`. *Espera:* dono — provisionamento do Vault. [registro](../registro/README.md)
- **1.3** `RTSK.024.1103.01` ✏️ Orquestrador `lock + rotate + verify + restart` por secret. *Aceite:* fluxo completo rodado em dry-run sem efeitos colaterais. *Entrega:* `internal/credrotator/orchestrator.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1101.01`, `RTSK.024.1102.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)

### Etapa 1.2 — Backends (Keycloak + Vault)

- **1.4** `RTSK.024.1104.01` ✏️ Atualizar `client_secret` via Keycloak admin API. *Aceite:* chamada bem-sucedida reflete-se em listagem via API; falha 4xx não corrompe estado. *Entrega:* `internal/credrotator/keycloak.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1101.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)
- **1.5** `RTSK.024.1105.01` ✏️ Gerar cryptorandom 32 bytes + escrever em Vault + arquivar versão anterior por 24h. *Aceite:* nova versão gravada, versão anterior lida por sample reader, auto-expira em 24h. *Entrega:* `internal/credrotator/vault.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1102.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)
- **1.6** `RTSK.024.1106.01` ✏️ Verificar alcance da nova secret via role de leitura amostra. *Aceite:* read role lê a nova versão antes de confirmar sucesso. *Entrega:* passo em `orchestrator.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1104.01`, `RTSK.024.1105.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)

### Etapa 1.3 — Rollout + modo de operação

- **1.7** `RTSK.024.1107.01` ✏️ Rolling restart via `kubectl rollout restart` após rotação. *Aceite:* restart aplicado só nos consumidores do secret (lista no catálogo); configurável per-secret. *Entrega:* passo em `orchestrator.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1106.01`. *Espera:* Baremetal — kubeconfig do runner. [registro](../registro/README.md)
- **1.8** `RTSK.024.1108.01` ✏️ Modo `--dry-run` executa tudo exceto write no IdP. *Aceite:* dry-run gera plano completo e não altera KeycloakOS/Vault. *Entrega:* flag em `cmd/cred-rotator/main.go`. *Escopo:* `cmd/cred-rotator/`. *Depende:* `RTSK.024.1103.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.4 — Auditoria + alerta

- **1.9** `RTSK.024.1109.01` ✏️ Emitir `audit_event(type='SECRET_ROTATED', secret_id, previous_hash, new_hash, rotated_at, actor='cred-rotator')`. *Aceite:* um evento por rotação, hash SHA256. *Entrega:* publisher em `orchestrator.go`. *Escopo:* `internal/credrotator/`, `pkg/audit/`. *Depende:* `RTSK.024.1106.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)
- **1.10** `RTSK.024.1110.01` ✏️ Notificação Slack em sucesso + page em falha. *Aceite:* mensagem em `#exchangeos-security` por sucesso; PagerDuty incident por falha. *Entrega:* integração em `orchestrator.go`. *Escopo:* `internal/credrotator/`. *Depende:* `RTSK.024.1103.01`. *Espera:* dono — credenciais Slack + PagerDuty. [registro](../registro/README.md)
- **1.11** `RTSK.024.1111.01` ✏️ Métrica `secret_rotation_last_success_timestamp{secret}`. *Aceite:* métrica exposta, 14 series ativas. *Entrega:* instrumentação em `cmd/cred-rotator/`. *Escopo:* `cmd/cred-rotator/`. *Depende:* `RTSK.024.1109.01`. *Espera:* sessão dona — implementação. [registro](../registro/README.md)

### Etapa 1.5 — Deploy + rollback

- **1.12** `RTSK.024.1112.01` ✏️ Helm CronJob `cred-rotator-cronjob.yaml` agendado mensal (dia 1, 03:00 UTC). *Aceite:* manifesto renderiza, agenda aplica. *Entrega:* `deploy/helm/exchangeos/templates/cred-rotator-cronjob.yaml`. *Escopo:* `deploy/helm/exchangeos/`. *Depende:* `RTSK.024.1103.01`. *Espera:* Baremetal — pipeline do Helm. [registro](../registro/README.md)
- **1.13** `RTSK.024.1113.01` ✏️ Script `scripts/cred-rollback.sh <secret>` para rollback emergencial. *Aceite:* script restaura versão anterior em < 1 min. *Entrega:* `scripts/cred-rollback.sh`. *Escopo:* `scripts/`. *Depende:* `RTSK.024.1105.01`. *Espera:* sessão dona — implementação; revisor: IdentityOS (credencial M2M). [registro](../registro/README.md)
- **1.14** `RTSK.024.1114.01` ✏️ Entrada de runbook cobrindo rollback emergencial. *Aceite:* runbook aponta para 1.13 e explica fluxo 4-eyes. *Entrega:* seção em `docs/operations/runbook-cred-rotator.md`. *Escopo:* `docs/operations/`. *Depende:* `RTSK.024.1113.01`. *Espera:* jurídico — revisão de procedimento emergencial. [registro](../registro/README.md)

### Etapa 1.6 — Testes

- **1.15** `RTSK.024.1115.01` ✏️ Teste de integração contra testcontainer KeycloakOS + Vault dev-server. *Aceite:* teste de rotação de 1 secret, idempotente em re-run, escrito; `mudanca` (N1) unitária verde (ADR-039); a rodada contra os testcontainers fica no fechamento do plano. *Entrega:* `tests/integration/cred_rotator_test.go`. *Escopo:* `tests/integration/`. *Depende:* `RTSK.024.1106.01`. *Espera:* gate — N1 unitária. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1116.01` ✏️ Rodar aceite da MS em staging. *Aceite:* todas as tarefas da Fase 1 cumpridas + 1 rotação mensal real em staging sem incidente + rollback testado uma vez. *Entrega:* relatório em `registro/ms-024c/`. *Escopo:* módulo inteiro + staging. *Espera:* gate — rodada `lote` do head de integração. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1117.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog. *Entrega:* commit de fechamento. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `docs/integrations/identityos.md` — catálogo dos 14 secrets
- `docs/security/iso27001-controls-mapping.md` controles 5.16, 5.17, 8.5
- `docs/security/sod-matrix.md` — papel do ator de rotação automática
