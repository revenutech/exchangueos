# Integração mobile — o que este módulo executa

Espelho das tarefas planejadas em `onboardos-revenu/.base/plans/` (programas `integracao-mobile-seguranca`, `integracao-mobile-tempo-real`, `limites-de-operacao`) cujo artefato de *Aceite* está neste repositório.
O plano de origem é a fonte do texto; esta lista é a fonte de quem executa. A trava é
`onboardos-revenu/scripts/plans/espelhos.py` (decisão C20 de 2026-10-23).

**Origem:** `onboardos-revenu#185` (na `develop`), `#190` (limites) e `#203` (cobertura), abertos em 2026-10-23. Quando entrarem, o texto de cada linha é o do plano de origem; o estado e o PR são deste repositório.

| tarefa | origem | o que entrega | estado | PR |
|---|---|---|---|---|
| `RTSK.080.804.01` | `onboardos:MS-163` 2.1 | Tirar o que é falso: `deploy/kafka/topics.yaml:100-115` deixa de declarar o `wss-gateway` como consumidor de `/ws/exchange/*`, que não existe | registrado | — |
| `RTSK.080.805.01` | `onboardos:MS-163` 2.2 | Outbox → Kafka para `quote.*` e `trade.*` no lugar do eventbus in-process; evento de cotação expirada (`RFQ.Expire()` com chamador); `trade.rejected`; client… | registrado | — |
| `RTSK.080.823.01` | `onboardos:MS-163` 2.3 | Contrato de produto do câmbio de varejo antes da 2.2: quem é a contraparte do cliente, que par de moedas, limites e o que o exchangeos (hoje interbancário) e… | registrado | — |
| `RTSK.080.824.01` | `onboardos:MS-163` 2.4 | Produto de câmbio de varejo da exchangeos ADR-009: `TradeKind{INTERBANK,RETAIL}`, `GET /pairs`, `POST /quote`, `POST /trades`, `GET /trades/{id}` com a ident… | registrado | — |
| `RTSK.080.917.01` | `onboardos:MS-165` 9.1 | Autenticação no módulo antes de qualquer exposição: interceptor JWKS no gRPC, middleware no HTTP, os dois «smoke endpoints» retirados e o CRUD admin nunca em… | registrado | — |
| `RTSK.080.918.01` | `onboardos:MS-165` 9.2 | Prova de IDOR no exchangeos: a operação é do tenant e do cliente do token | registrado | — |
| `RTSK.008.514.01` | `onboardos:MS-201` 3.5 | Câmbio de varejo: o teto regulatório por operação (eFX US$ 10 mil sem fracionar; o de US$ 100 mil da Res. BCB 277 art. 29 só se o produto sair do eFX, em ADR… | registrado | — |
