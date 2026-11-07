# Registro de milestones — ExchangeOS

Caminho canônico citado em `../CONVENTIONS.md` § 6.2 do baremetal (CONVENTIONS 1.6.0): a linha de cada tarefa fica curta (~400 caracteres, só estado, aceite, entrega, escopo, depende, espera e link); **a narrativa e a evidência vivem aqui**.

## Organização

```
registro/
├── README.md                          (este arquivo)
├── ms-NNNx/
│   ├── plano-original-shape-1.1.0.md  (texto pré-migração, quando aplicável)
│   └── YYYY-MM-DD-<tarefa-slug>-<4hex>.md
│                                      (um por evento, gerado por scripts/registro/registro.py novo
│                                       assim que a Baremetal publicar o script no exchangeos)
```

## Por que preservar o `plano-original-shape-1.1.0.md`

Em 2026-11-07 o dono decidiu migrar `active/` e `backlog/` para CONVENTIONS 1.6.0 (ordem via Core, select registrada no Core em 2026-11-07). A reescrita em shape nova é completa — campos, estado por linha, Fases, Etapas, link para `registro/`. Para que o texto original de cada aceite não suma da leitura, cada MS ajustada tem o seu texto pré-migração preservado neste diretório, na shape 1.1.0 que valia até então. A migração está citada no campo `**Fonte original:**` do cabeçalho de cada MS do backlog.

Esta pasta **não substitui** `delivered/`: ela guarda evidência e narrativa das MS que ainda estão em backlog/ ou active/. Quando uma MS entra em `delivered/`, o seu subdiretório aqui permanece com o histórico de execução.

## Do campo `evidencia` do registro de execução (§ 6.7)

Quando o passo de uma Etapa fechar verde, o registro do passo carrega:

```
evidencia:
  head:    abc1234
  entrada: go.sum, migrations/
  hash:    9f2c
```

`entrada` é o que o passo lê (caminhos, versões, dados de partida). O `hash` da entrada é a chave de idempotência — só se re-roda o passo cuja entrada mudou.

## Pendências

- `scripts/registro/registro.py novo` não existe no exchangeos; é esperado da Baremetal (CONVENTIONS 1.7.0+). Até lá, novos registros são criados à mão seguindo a convenção `YYYY-MM-DD-<slug>-<4hex>.md`.
- Códigos `RMLS.024.FFF.VV` e `RTSK.024.FFF.VV` ainda carregam placeholder `???` — a Baremetal vai passar o M numérico das sub-letras (b, c, d, f, g, i, j, k, m) depois da decisão do dono em 2026-11-07 de registrar o domínio 024 e alocar M.
