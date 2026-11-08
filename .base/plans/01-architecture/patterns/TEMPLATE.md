# Pattern Template — ExchangeOS

> Template obrigatório para toda entrada nova de `patterns/` (MS-024k 1.1, `RTSK.024.1701.01`).
> Fonte da forma: template implícito citado em [`200-fx-golang-patterns.md`](200-fx-golang-patterns.md) (seção «Pattern template»), agora formalizado aqui e referenciado pelas demais famílias (`FX-DDD-*`, `FX-EDA-*`, `FX-CP-*`, `FX-COMMIT-*`, `FX-QA-*`, `FX-CRDB-*`, `FX-KAFKA-*`, `FX-DEVSECOPS-*`).

## Como usar

1. Copie o bloco «Forma» abaixo para dentro do catálogo da família (ex.: `200-fx-golang-patterns.md`, `201-fx-ddd-patterns.md`, …), logo na tabela de catálogo e com uma seção H2 por `FX-<família>-NNN`.
2. Preencha as **6 seções obrigatórias** na ordem — `Context`, `Problem`, `Solution`, `Example`, `Anti-pattern`, `Related` — cada uma com no máximo cerca de 400 palavras (ratchet do § 6.5 da CONVENTIONS). Se crescer além disso, divida em dois padrões.
3. `Example` deve apontar um caminho real de produção (`<path>:<linha>` ou função nomeada do repositório); `Related` deve citar ≥ 1 outro padrão (`FX-<família>-NNN`) ou outro documento do catálogo (RN_FX_*, FX-DDD-*, etc.).
4. O lint da MS-024k 1.2 (`scripts/lint-patterns.sh`, ainda pendente) valida este contrato; até ele existir, a revisão é humana.

## Forma

````markdown
### FX-<FAM>-NNN — <título curto e específico>

**Context:** situação em que o padrão se aplica — stack, camada (domain/application/infrastructure), ponto no ciclo (construção, persistência, publicação, chamada externa). Uma a três frases, sem jargão sem origem.

**Problem:** o que dá errado sem o padrão. Cite o custo concreto (teste frágil, regra de negócio bypass, latência, acoplamento) e por que é não-óbvio. Uma a três frases.

**Solution:** a forma canônica. Prefira linguagem ubíqua do domínio (`FXTrade`, `FXQuote`, `PvP`, `PTAX`, `DEC`) a substantivos genéricos. Diga explicitamente o que é imutável (campos privados, estado interno, contrato gRPC v1).

**Example:**

```go
// caminho real de produção; sem pseudo-código
func NewFXTrade(in NewTradeInput) (*FXTrade, error) {
    if err := in.validate(); err != nil {
        return nil, err
    }
    // ...
    return t, nil
}
```

Cite a origem: `modules/trade/domain/fxtrade.go:NewFXTrade`. Para padrão não-Go (Flink job, SQL migration, Helm template, proto), o bloco cercado usa a linguagem apropriada (`sql`, `yaml`, `proto`, `hcl`, `python`, `bash`) e aponta o arquivo real.

**Anti-pattern:** o que NÃO fazer, com uma frase explicando por que falha na prática. Não é lista genérica de «más práticas» de livro — é a tentação concreta que aparece no código do ExchangeOS.

**Related:** `FX-<FAM>-MMM`, `FX-<OUTRA>-NNN`, `RN_FX_NNN` ou documento de catálogo relevante. Sem o campo, o padrão fica isolado; a MS-024k 1.11 exige ≥ 1 cross-link.
````

## Regras de forma

- **6 seções obrigatórias**, nessa ordem; omitir qualquer uma reprova a entrada.
- **≤ ~400 palavras por seção.** Se o padrão precisa de muito mais, há outro padrão escondido — divida.
- **1 exemplo mínimo de código de produção.** Sem `foo/bar` nem `TODO`. O caminho real com `<path>:<linha>` (ou função nomeada) é o que amarra padrão a repositório vivo.
- **1 anti-pattern.** O «NÃO faça» que você já viu no código real do módulo, não folclore.
- **1 cross-link no `Related`.** Padrão isolado é dívida de catálogo.

## Nota sobre números

Os identificadores `FX-<FAM>-NNN` são imutáveis uma vez publicados. Padrão retirado permanece com o número e recebe sinal `⏸️ retirado em <data>: <motivo>` na linha da tabela de catálogo (nunca é reutilizado), análogo ao que a CONVENTIONS do baremetal faz com código `RTSK` arquivado.

## Referências

- `.base/plans/01-architecture/patterns/README.md` — índice dos 20 catálogos e contagem viva.
- `.base/plans/01-architecture/patterns/200-fx-golang-patterns.md` — primeira referência de uso (FX-GP-001 a FX-GP-010 completos).
- `baremetal/.base/plans/CONVENTIONS.md` § 6.2, § 6.5 — ratchet de ~400 caracteres por linha, forma e evidência.
- `CLAUDE.md` workflow §2 (TDD-first) — contexto de consumo dos padrões.
- ISO 27001 controle 5.37 (documented operating procedures).
