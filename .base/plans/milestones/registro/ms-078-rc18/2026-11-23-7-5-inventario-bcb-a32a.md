---
data: 2026-11-23
tarefa: 7.5
estado: 🔄
evidencia: inventário por leitura do código em develop c6758ba (exchangeos); consolidação na Core
pr: —
---
MS-078 7.5 (RC18-T05, P0, do ledgeros): inventário das remessas ao BCB, parte do **exchangeos**. Divisão por módulo feita pela Core; o dono ativou a tarefa em 2026-11-23. O inventário trata do que o exchangeos **gera para o banco (tenant)** remeter ao BCB, não do que a Revenu remete (DV 2875/2879). Feito só por leitura: sem código e sem PII.

**Hipótese** (anexo T do RPLN.050.001.01 no ledgeros, linha 102, [mat] matriz-modulos.md): o exchangeos produz o registro de operações de câmbio. **Resultado: a hipótese cai em grande parte.** O exchangeos não monta o conteúdo de nenhuma remessa. Ele só tem um rastreador de envio (`BACENReport`) e um envelope XML de metadados, e nada é transmitido.

Marcas: [M] = medido no código ou nos testes; [I] = inferido de plano ou documento.

| remessa/documento | leiaute (código BCB) | periodicidade e prazo | origem do dado | gera hoje? | evidência | hipótese |
|---|---|---|---|---|---|---|
| Registro de câmbio (`BCB-CAMBIO`) | sem código: `<BACENReport>` é envelope próprio, «não é schema do BACEN nem da ISO» | não declarada | tabela `bacen_reports` (`migrations/000008_create_compliance_admin.up.sql:42`); RPC `SubmitBACENReport` (`proto/exchangeos/v1/compliance.proto:15`); o conteúdo chega **pronto, em bytes opacos, de quem chama**, e só o sha256 fica gravado | parcial (rastreador e envelope; não monta conteúdo nem envia) | `pkg/bacen/submission/marshal.go:23-31` [M]; `modules/compliance/application/service.go:293-304` [M]; `modules/compliance/api/grpc_server.go:121-130` [M] | derruba |
| SISBACEN (EOD diário) | sem código | diária, no EOD [I] | mesma tabela e RPC; fluxo RFLW.024.050.01 | não (o EOD é um TODO) | `cmd/eod/main.go:31` [M]; `.base/flows/eod/RFLW.024.050.01.md:52` [I] | nova |
| `BCB-CCS` | sem código | não declarada | mesma tabela e RPC | parcial (só o envelope) | `modules/compliance/domain/bacen_report.go:24-28` [M] | nova |
| DEC (Lei 14.286/2021) | Kind `DEC`, sem XSD | não declarada | nenhuma: os `Kind` não mapeiam para os `ReportType` do domínio | não (interface e mock; `pkg/bacen/dec.go` não existe) | `pkg/bacen/submission/submitter.go:26-30` [M]; `.base/plans/milestones/backlog/MS-024f-bacen-submission-adapters.md:28` [I] | nova |
| SCE-IED | Kind `SCE-IED` | «registro em 30 dias» [I], sem norma citada | nenhuma | não | `submitter.go:27` [M]; MS-024f:29 [I] | nova |
| SCE-Crédito (sucessor do RDE-ROF) | Kind `SCE-CREDITO` | não declarada | nenhuma | não | `submitter.go:28` [M]; MS-024f:30 [I] | nova |
| SCE-CBE | Kind `SCE-CBE` | não declarada | nenhuma | não | `submitter.go:29` [M]; MS-024f:31 [I] | nova |
| COS ao SISCOAF/COAF | «XSD COAF 2026» planejado; `pkg/siscoaf` não existe | 1 dia útil após a decisão (RN_FX_039, PROPOSED) [I] | `cos_cases` (`migrations/000011_create_cos_cases.up.sql:32`); eventos `EventCOSCaseSubmitted/Accepted/Rejected`; gatilho `ScreeningResult.RequiresCOS` | não (agregado com 4-eyes; sem serviço, repositório, API, XML nem transporte) | `modules/compliance/domain/cos_case.go:36-38,185` [M]; `.base/plans/milestones/backlog/MS-024g-siscoaf-cos-submission.md:37-38` [I] | nova |
| IOF | sem leiaute (só cálculo) | não declarada | `iof_computations` (000008:25); RPC `ComputeIOF` | não (calcula o imposto e não gera remessa) | `pkg/bacen/iof.go:10-19` [M] | nova |
| Sistema Câmbio / SISCOMEX | — | — | — | não (só citados em listas de plano) | `.base/plans/09-compliance/index.md:48` [I] | — |

Fora do inventário: PTAX/OLINDA é dado que o exchangeos **lê** do BCB, não remessa. Não há códigos CADOC (2011/2060/3040/4111/6334), mensagens STR/Tx 70, ROC, arquivos `.xsd` nem os diretórios `api/openapi` e `api/asyncapi`.

## O que derruba a hipótese

- **O dado do registro não está no exchangeos.** `fx_trades` (`migrations/000002`) não tem natureza nem CPF/CNPJ. Por operação, o compliance só guarda a natureza (`classifications`) e o IOF (000008:8,25).
- **O conteúdo vem de fora.** O `SubmitBACENReport` recebe o payload pronto, em bytes opacos, e guarda só o hash (`grpc_server.go:121-130`).
- **O envio é de outro módulo, por desenho.** Segundo `docs/integrations/authorityos.md:9-13,32-44`, a AuthorityOS submete ao BACEN/SISCOAF quando sai o evento `compliance.report_ready.v1`, e nenhum código do exchangeos emite esse evento [M]. O mesmo documento afirma que o exchangeos «generates raw BACEN reports» (linhas 10-11) [I], mas o código não gera: o documento promete o que não existe, assim como o MS-024f.
- **Os dois conjuntos de tipos não conversam.** O domínio usa SISBACEN, BCB-CCS e BCB-CAMBIO; o submitter usa DEC e SCE-*. O próprio repo deixa isso em aberto (`registro/ms-024f/2026-11-10-marshal-7b2a.md:13`).
- **O plano promete mais do que o código tem.** O MS-024f:10 diz que as classes DEC e SCE-* já existem em `modules/compliance/domain`, e elas não existem [M].
- **Comentário falso no código.** O comentário de `service.go:293` diz «marks SUBMITTED», mas o relatório fica PENDING [M].

## Em aberto

1. **Quem gera o conteúdo** do registro de câmbio, da DEC e das SCE-*: o exchangeos, a AuthorityOS ou o core do banco? (dono)
2. **O que é cada tipo do domínio:** `BCB-CCS` é o Cadastro de Clientes do SFN? Que documento concreto é o «SISBACEN»? (dono, contador)
3. **A regra da DEC:** «DEC > USD 10K» vale para a mesa de câmbio do banco, pela Lei 14.286 e pela Res. BCB 277? (jurídico)
4. **Leiautes, versões e prazos** de cada tipo, inclusive os 30 dias da SCE-IED: o repo não cita norma de prazo além da RN_FX_039. (contador, jurídico)
5. **Gatilho e especificação da COS:** um acerto HIGH em sanções obriga COS? Que XSD do COAF vale? (jurídico)
6. **Obrigação de IOF do módulo:** cabe ao módulo de câmbio alguma declaração do IOF recolhido pelo banco? (contador)
7. **Códigos de natureza:** há 46 códigos no código e 95 citados no `doc.go`, e o mesmo vale para o glossário (`codes_full.go:4`). Basta a natureza para o registro? (contador)

Assinado: sessão ExchangueOS, 2026-11-23, para a consolidação da Core.
