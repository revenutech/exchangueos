# Eventos — exchangeos

<!-- GERADO por `msgrepo sync exchangeos`. NÃO EDITE.
     A fonte é ledgeros/.base/aasc/iso20022/. Editar aqui é erro de build
     (RN_MSG_013), não decisão de estilo: esta pasta é projeção, e uma
     projeção editável é a segunda fonte de verdade de volta. -->

| | |
|---|---|
| Emite | 0 mensagem(ns) |
| Consome | 0 mensagem(ns) |

## Business areas

_Nenhuma._

## Como isto se mantém em dia

```bash
msgrepo sync exchangeos          # regenera
msgrepo sync exchangeos --check  # o que o CI roda
```

O emissor não aparece no identificador: vive no BAH `head.001`, campo `Fr`
(RN_MSG_003). Uma mensagem listada aqui pode ter outros emissores — a
definição pertence ao fato, não ao módulo (RN_MSG_004).
