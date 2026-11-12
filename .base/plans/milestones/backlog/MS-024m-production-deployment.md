# MS-024m — Production Deployment (GKE + Litmus + Chaos Mesh + ArgoCD live)

**Code:** `RMLS.024.018.01` M alocado pela Baremetal em 2026-11-07 (§ 3.3.5 da CONVENTIONS, em `main` desde a 1.9.0): M=18; registro do DDD 024 em curso.
**Status:** BACKLOG
**Owner:** Platform
**Fonte original (shape 1.1.0, preservada em 2026-11-07):** [registro/ms-024m/plano-original-shape-1.1.0.md](../registro/ms-024m/plano-original-shape-1.1.0.md)

## Por quê

Todos os artefatos de deploy existem (Helm + Terraform GKE + ArgoCD Application + Argo Rollouts canary + cert-manager + SLSA L3 + Vault seed + manifestos de chaos). Nenhum aplicado a cluster real. Sem isso, o sistema é shelf-ware. Esta MS fecha o gap de "design done" para "running production with traffic".

## Descrição

Provisionar projeto GCP de produção + cluster GKE Autopilot + infra de apoio via Terraform, deploy via ArgoCD pull, rodar smoke + load + chaos contra staging primeiro, promover a produção via canary. Inclui install de Litmus + Chaos Mesh para o programa de chaos (MS-024k indireto).

## Fase 1 — Implementação

### Etapa 1.1 — Infra (Terraform)

- **1.1** `RTSK.024.1801.01` ✏️ `terraform apply` sobre `deploy/terraform/environments/production/`. *Aceite:* apply termina sem drift. *Entrega:* state em workspace. *Escopo:* `deploy/terraform/environments/production/`. *Espera:* dono — provisionamento do projeto GCP. [registro](../registro/README.md)
- **1.2** `RTSK.024.1802.01` ✏️ Cluster GKE Autopilot 1.29+ com WIF enabled. *Aceite:* `kubectl cluster-info` responde + WIF pool configurado. *Entrega:* módulo Terraform. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* Baremetal — infra. [registro](../registro/README.md)
- **1.3** `RTSK.024.1803.01` ✏️ VPC + Cloud NAT + private cluster + master authorized networks restringido. *Aceite:* `terraform plan` zero diffs, acesso ao master apenas da bastion. *Entrega:* módulo Terraform. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* Baremetal — infra. [registro](../registro/README.md)
- **1.4** `RTSK.024.1804.01` ✏️ CMEK em etcd via Cloud KMS HSM. *Aceite:* chave HSM criada, GKE usa CMEK. *Entrega:* módulo Terraform. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* dono — definição da política de chave; revisor: IdentityOS (assinatura: chave). [registro](../registro/README.md)
- **1.5** `RTSK.024.1805.01` ✏️ Binary Authorization policy aplicada (só imagens assinadas). *Aceite:* imagem não-assinada é rejeitada em teste. *Entrega:* `deploy/terraform/binauthz.tf`. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* dono — definição do signatário; revisor: IdentityOS (assinatura). [registro](../registro/README.md)
- **1.6** `RTSK.024.1806.01` ✏️ Bucket GCS archive com lifecycle policy. *Aceite:* bucket criado com lifecycle de 2 anos (Nearline → Coldline → Archive). *Entrega:* `deploy/terraform/storage.tf`. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* Baremetal — infra. [registro](../registro/README.md)
- **1.7** `RTSK.024.1807.01` ✏️ Budget module aplicado com threshold de produção. *Aceite:* alerta em 80% + 100% configurado. *Entrega:* `deploy/terraform/budgets.tf`. *Escopo:* `deploy/terraform/`. *Depende:* `RTSK.024.1801.01`. *Espera:* dono — definição de thresholds. [registro](../registro/README.md)

### Etapa 1.2 — Bootstrap do cluster

- **1.8** `RTSK.024.1808.01` ✏️ cert-manager + ClusterIssuer LE (prod+staging). *Aceite:* 1 certificado emitido em cada issuer. *Entrega:* manifestos em `deploy/bootstrap/`. *Escopo:* `deploy/bootstrap/`. *Depende:* `RTSK.024.1802.01`. *Espera:* Baremetal — pipeline do bootstrap. [registro](../registro/README.md)
- **1.9** `RTSK.024.1809.01` ✏️ ArgoCD + `application.yaml` + `AppProject`. *Aceite:* ArgoCD UI acessível, project criado. *Entrega:* `deploy/argocd/`. *Escopo:* `deploy/argocd/`. *Depende:* `RTSK.024.1802.01`, `RTSK.024.1808.01`. *Espera:* Baremetal — pipeline do bootstrap. [registro](../registro/README.md)
- **1.10** `RTSK.024.1810.01` ✏️ Argo Rollouts controller. *Aceite:* controller rodando, CRDs disponíveis. *Entrega:* manifesto em `deploy/argocd/`. *Escopo:* `deploy/argocd/`. *Depende:* `RTSK.024.1809.01`. *Espera:* Baremetal — pipeline do bootstrap. [registro](../registro/README.md)
- **1.11** `RTSK.024.1811.01` ✏️ Litmus + Chaos Mesh em namespaces dedicados com RBAC restrito a `chaos-runner` SA. *Aceite:* chaos SA pode listar workloads, nenhuma outra. *Entrega:* `deploy/chaos/`. *Escopo:* `deploy/chaos/`. *Depende:* `RTSK.024.1802.01`. *Espera:* dono — aprovação do programa de chaos. [registro](../registro/README.md)
- **1.12** `RTSK.024.1812.01` ✏️ External Secrets Operator + Vault auth/kubernetes bind. *Aceite:* 1 secret externa referenciada por ESO rende em ConfigMap/Secret. *Entrega:* `deploy/eso/`. *Escopo:* `deploy/eso/`. *Depende:* `RTSK.024.1802.01`. *Espera:* Baremetal — Vault de produção; revisor: IdentityOS (credencial). [registro](../registro/README.md)
- **1.13** `RTSK.024.1813.01` ✏️ Vault produção alcançável + `scripts/vault-seed.sh` executado para path prod. *Aceite:* 14 secrets do catálogo gravadas em prod path. *Entrega:* execução de `scripts/vault-seed.sh prod`. *Escopo:* `scripts/`. *Depende:* `RTSK.024.1812.01`, `RTSK.024.1101.01`. *Espera:* dono — provisionamento do Vault de produção; revisor: IdentityOS (credencial). [registro](../registro/README.md)

### Etapa 1.3 — Workload

- **1.14** `RTSK.024.1814.01` ✏️ ArgoCD sync da Application ExchangeOS — todos os pods Ready. *Aceite:* sync verde, `kubectl get pods` 100% Running. *Entrega:* evento ArgoCD. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1809.01`, `RTSK.024.1813.01`. *Espera:* Baremetal — ArgoCD. [registro](../registro/README.md)
- **1.15** `RTSK.024.1815.01` ✏️ Helm release `exchangeos` deployado (api + worker + migrator job + cls-cycle + eod + mq-bridge + cred-rotator). *Aceite:* 7 componentes Running/Completed. *Entrega:* Helm release. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1814.01`. *Espera:* Baremetal — ArgoCD. [registro](../registro/README.md)
- **1.16** `RTSK.024.1816.01` ✏️ Migrator job completo para migrações 000001-current. *Aceite:* `migrator` termina com código 0, versão bate. *Entrega:* evento de job. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1815.01`. *Espera:* gate — migrações. [registro](../registro/README.md)
- **1.17** `RTSK.024.1817.01` ✏️ 14 serviços BC respondem a gRPC + REST health probes. *Aceite:* `/healthz` e `/readyz` 200 em 14 BCs. *Entrega:* evidência de probes. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1815.01`. *Espera:* sessão dona — validação. [registro](../registro/README.md)
- **1.18** `RTSK.024.1818.01` ✏️ Prometheus raspa todos targets + Grafana dashboards (delivery + SLO + FinOps) com dados. *Aceite:* 3 dashboards com séries ativas. *Entrega:* evidência de dashboard. *Escopo:* observabilidade. *Depende:* `RTSK.024.1817.01`. *Espera:* sessão dona — validação. [registro](../registro/README.md)

### Etapa 1.4 — Validação

- **1.19** `RTSK.024.1819.01` ✏️ `task smoke:prod` retorna 0 (7 checks pass). *Aceite:* script completo verde. *Entrega:* log em `.audit-bundles/prod-bringup-YYYYMMDD/smoke.log`. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1817.01`. *Espera:* gate — smoke. [registro](../registro/README.md)
- **1.20** `RTSK.024.1820.01` ✏️ `k6 run tests/load/k6-trade-book.js` contra staging atende SLO. *Aceite:* p95 trade-book < limite SLO. *Entrega:* relatório k6. *Escopo:* staging. *Espera:* gate — load. [registro](../registro/README.md)
- **1.21** `RTSK.024.1821.01` ✏️ 1 canary rollout 10→30→60→100 com AnalysisTemplate Prometheus verde. *Aceite:* rollout completa sem rollback automático. *Entrega:* evento ArgoRollouts. *Escopo:* cluster de produção. *Depende:* `RTSK.024.1815.01`. *Espera:* gate — canary. [registro](../registro/README.md)
- **1.22** `RTSK.024.1822.01` ✏️ CHAOS-01 + CHAOS-02 em staging via Litmus. *Aceite:* 2 experimentos completam sem derrubar SLO. *Entrega:* relatórios Litmus. *Escopo:* staging. *Depende:* `RTSK.024.1811.01`. *Espera:* gate — chaos. [registro](../registro/README.md)
- **1.23** `RTSK.024.1823.01` ✏️ Drill de restore-from-backup + RTO/RPO medidos. *Aceite:* restore ≤ 30 min (RTO) e perda ≤ 1 min (RPO), 0 para liquidação aceita (ADR-038 classe 2). *Entrega:* relatório em `.audit-bundles/`. *Escopo:* DR. *Depende:* `RTSK.086.752.01`. *Espera:* gate — DR drill (alvo do ADR-038 `RADR.076.002.01`, Aceito em 2026-11-12). [registro](../registro/README.md)

### Etapa 1.5 — Handover operacional

- **1.24** `RTSK.024.1824.01` ✏️ On-call enrolled em PagerDuty `exchangeos-api-prod`. *Aceite:* rotation configurada, 1 teste de alerta recebido. *Entrega:* configuração PagerDuty. *Escopo:* PagerDuty. *Espera:* dono — definição da rotation. [registro](../registro/README.md)
- **1.25** `RTSK.024.1825.01` ⬜ Runbook index validado (todos os links resolvem). *Aceite:* verificador link-checker 0 falhas em `docs/operations/`. *Entrega:* evidência. *Escopo:* `docs/operations/`. *Espera:* sessão dona — validação. [registro](../registro/README.md)
- **1.27** `RTSK.024.1829.01` ✏️ Rodar de novo o link-checker de `docs/operations/` perto do go-live (a 1.25 roda agora, por decisão do dono em 2026-11-12). *Aceite:* 0 falhas na semana do go-live. *Entrega:* evidência. *Escopo:* `docs/operations/`. *Depende:* `RTSK.024.1825.01`. *Espera:* dono — data do go-live. [registro](../registro/README.md)
- **1.26** `RTSK.024.1826.01` ✏️ Primeiro chaos day agendado + primeira revisão trimestral de custo + primeira revisão LGPD + primeiro DR drill. *Aceite:* 4 eventos no calendário com responsável. *Entrega:* convites de calendário. *Escopo:* calendário. *Espera:* dono — agenda. [registro](../registro/README.md)

## Fase 2 — Aceite

- **2.1** `RTSK.024.1827.01` ✏️ Rodar aceite da MS. *Aceite:* todas as tarefas da Fase 1 cumpridas + `docs/operations/go-live-checklist.md` com todas as linhas ✅ assinadas + sistema serve 1 trade real. *Entrega:* `.audit-bundles/prod-bringup-YYYYMMDD/` completo + checklist. *Escopo:* módulo inteiro + produção. *Espera:* gate — rodada N2 + canary. [registro](../registro/README.md)

## Fase 3 — Fechamento

- **3.1** `RTSK.024.1828.01` ✏️ Mover MS para `delivered/` e atualizar `milestones/index.md`. *Aceite:* MS em `delivered/`, índice sem referência em backlog, newsletter interna publicada. *Entrega:* commit de fechamento + post. *Escopo:* `.base/plans/milestones/`. *Espera:* dono — aprovação final. [registro](../registro/README.md)

## Referências

- `docs/operations/go-live-checklist.md` — checklist primário que esta MS fecha
- `docs/operations/canary-runbook.md` — procedimento canary
- `docs/security/dr-runbook.md` — restore drill
- `docs/security/chaos/chaos-day-runbook.md` — chaos day kickoff
- Todas as MS-024* anteriores (maioria é pré-requisito)
- Critério de sucesso: sistema serve trades reais
