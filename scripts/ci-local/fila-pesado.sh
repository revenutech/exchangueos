#!/usr/bin/env bash
# Fila do gate pesado da máquina, por ordem de chegada (dono, 2026-10-27, O194: «Fila com senha»).
# Gate pesado = CRDB, Kafka, e2e ou `go test -race` longo.
# Versionado aqui (dono, select no Core em 2026-11-02 ~08:40): /private/tmp/revenu-gate-pesado.sh é um LINK para este
# arquivo, feito por instalar-fila.sh; só o estado (senhas, vagas) fica no /tmp. O /tmp perde arquivo na limpeza do macOS.
# Vagas (dono, 2026-10-27; ajuste do Core às 10:35 para o M5 Max de 18 núcleos): até 5 vagas (dono, select no Core às ~10:55: «5 vagas») com carga de 1 min < 54, caindo em degraus até
# 1 perto de 90, e nenhuma NOVA a partir de 90 (quem já roda continua). Timeout de 90 min por gate no modo comando.
#
#   revenu-gate-pesado.sh <dono> <comando…>     → espera a vez, roda o comando e solta a vaga (fim, falha, sinal ou timeout)
#   . revenu-gate-pesado.sh; fila_entrar <dono>; …; fila_sair   → o mesmo, por dentro de outro script (sem timeout)
#   revenu-gate-pesado.sh --nivel <base> [<head>] [--previsto <arquivo>]   → só lê: N0, N1 com os pacotes (um go.mod ou
#       vários), ou amplia seguido de uma ou mais linhas «motivo: …», qualquer que seja o gatilho (CONVENTIONS § 6.10)
#   revenu-gate-pesado.sh --hash <passo> <caminho…> → chave de idempotência da entrada (árvore, go.mod/go.sum, go, tools.lock);
#       --prova <passo> <hash> [<caminho…>] (árvore suja não grava) e --reaproveita <passo> <hash>
#   revenu-gate-pesado.sh --tem-vaga leve|banco   → só lê: rc 0 se uma senha nova da pista entraria agora, 1 se não
#   revenu-gate-pesado.sh --status   → por senha viva: pista, senha, dono, pid, vaga ou espera, e o passo e o pacote em curso
#       lidos do progresso.tsv que a prova declarou em REVENU_GATE_PROGRESSO (CONVENTIONS § 6.6); ESPERA>30min e, com
#       REVENU_GATE_NIVEL_PROVA=N1|N2, TETO>N1 (15 min) ou TETO>N2 (45 min) de vaga (§ 6.10)
#   revenu-gate-pesado.sh --leituras   → portão: leitura de disco fora do pacote só com «le-disco <pacote> <glob>» no
#       .ci-local/escopo.conf (rc 1 sem); --nivel soma esses pacotes e --hash aceita «le-disco:<pacote>» (§ 6.10)
#   revenu-gate-pesado.sh --dlq | --dlq-tira <chave> | --breaker-fecha   → DLQ e breaker (CONVENTIONS § 6.10)
#   revenu-gate-pesado.sh --guardar-conteiner [<contêiner…>]   → log e .State de contêiner de gate que caiu, antes do rm
#
# Vaga 1 = /private/tmp/revenu-gate-pesado.lock (o mesmo lock de antes, com `pid` e `dono`); vaga N = <lock>.N.
# Quem ainda espera no laço antigo (mkdir do lock a cada 30 s) continua funcionando: só disputa a vaga 1.
# Senha: mkdir atômico de <senhas>/<n>, com n acima de todas as senhas vivas. A senha fica viva enquanto o gate roda
# (com o arquivo `vaga` depois que entra). Só a CABEÇA da fila (a menor senha viva que ainda espera) disputa vaga, e
# só ela lê a carga: o número de vagas não varia entre quem espera, e ninguém passa à frente de uma senha menor.
# Senha antiga (sem `cabeca`, do código anterior) que espera À FRENTE só disputa a vaga 1: enquanto houver uma,
# a vaga 1 fica para ela e a cabeça daqui usa as vagas 2 e 3 (Core, 2026-10-27 ~10:55 BRT: as senhas 11–36 esperavam
# desde 07:30 enquanto as novas passavam à frente, contra a O197).
# Senha de PID morto é apagada; senha sem pid (o dono ainda não gravou) vale por 60 s.
# Memória (dono, selects no Core em 2026-10-28, ~16:45 «vagas pela memória» e ~16:55 «warn abre até 2 vagas»): com swap
# acima de 85% E o kernel em pressão warn, rodam no máximo 2 gates juntos; em pressão critical, nenhuma vaga NOVA (a fila
# vazia ainda abre 1). Quem já roda continua. As duas leituras juntas porque o swap do macOS fica alto muito depois de a
# memória voltar. A carga de CPU 90 continua zerando, como antes.
# 2026-11-13 (dono, select na sessão Baremetal, revogação repassada pela Core): sai a regra das sessões «swap < 92% antes da
# N2 com banco»; quem entra na fila não confere o swap antes, e vale só esta regra (85% + warn → 2; critical → nenhuma nova).
# Reconstruído em 2026-11-02 pelo Core a partir da transcrição: o arquivo sumiu do /tmp (limpeza do macOS, provável).
# Ajustes por ambiente: REVENU_GATE_VAGAS_MAX (5), REVENU_GATE_CARGA_2 (54), REVENU_GATE_CARGA_0 (90), REVENU_GATE_TIMEOUT_S (5400),
# REVENU_GATE_SWAP_MAX (85, em %), REVENU_GATE_VAGAS_WARN (3), REVENU_GATE_CRDB_MAX (3), REVENU_GATE_LEVE_MAX (2), REVENU_GATE_LEVE_WARN (2).
# Rodada paralela: quem exporta REVENU_GATE_RODADA=1 não perde vaga por carga; memória e teto de CRDB seguem (dono, select no Core em 2026-11-07).
# Pista sem banco: quem exporta REVENU_GATE_SEM_BANCO=1 (lint, vet, unitário) tem senhas e vagas próprias e não espera CockroachDB.
# Nível unitário (dono, 2026-11-18; rodada-escopo.sh): RODADA_NIVEL=unitaria ou REVENU_GATE_NIVEL_TESTE=unitaria também
# entra na pista sem banco; a senha guarda o nível em `nivel_teste` (o `nivel` N1/N2 segue igual), o --status acrescenta
# «nível=unitária» ao estado e a prova gravada leva «nivel=unitaria».
# Dono, select no DV em 2026-11-06: até 3 gates com banco juntos, também em warn (revoga «um por vez» de 05/11); critical segue 1.
# Rodada pesada exclusiva e memória da VM do Docker (dono, sessão Baremetal em 2026-11-22, depois do incidente das 00:31
# BRT: a VM tem 15,6 GB para tudo — malha ~4,4 GB, e2e ~1,5 GB, act —, uma rodada com CockroachDB próprio junto de outra
# estourou a VM e os CockroachDB caíram juntos, Exited 7). (1) Pesada = toda rodada da pista banco, ou a da leve que
# exporta REVENU_GATE_PESADA=1 (mutation-gate/gremlins): roda sozinha, nas duas pistas; entra só com nenhuma outra
# rodando, e nenhuma entra enquanto ela roda ou espera só pela exclusiva. Vence o «até 3 com banco» de 06/11.
# (2) Nenhuma rodada nova, nas duas pistas, com menos de REVENU_GATE_MEM_LIVRE_MIN_MB (6144) livres na VM (MemTotal do
# docker info menos a soma do MemUsage do docker stats). Sem docker ou com leitura falha, não bloqueia: o --status e o
# log dizem «memória da VM: não lida». Quem já roda termina. REVENU_GATE_EXCLUSIVA=0 desliga (1), MEM_LIVRE_MIN_MB=0 (2).
# Correção do dono no mesmo dia: na pista leve, (2) vale só para quem declara contêiner (REVENU_GATE_DOCKER=1).
# «As duas, com teto» (dono, sessão Baremetal em 2026-11-22 ~19:00 BRT: uma pesada esperou 21 min pela exclusiva e
# 17 leves unitárias pararam atrás dela). (3) Pesada só com liberação: a DV 2397 diz que hoje vale só a N1 unitária;
# a senha pesada (pista banco, ou leve com REVENU_GATE_PESADA=1) sem REVENU_GATE_PESADA_LIBERADA=<motivo> no ambiente
# de quem pede é recusada na hora, rc 77, sem pegar senha; o motivo da liberação vai ao log. Desliga com a constante
# FILA_PESADA_LIBERACAO abaixo ou com PESADA_LIBERACAO=0 no .limites (o vigia reescreve o arquivo e a regra volta);
# nunca pelo ambiente de quem pede. (4) Janela da leve: enquanto uma pesada espera a exclusiva, a leve sem contêiner
# segue entrando como sempre (teto de vagas leves e memória mínima da VM), por até JANELA_LEVE_S (600) desde o pedido
# da exclusiva; depois, a fila drena e a pesada entra. Com contêiner, espera desde o início. O fim da janela fica na
# senha da pesada (janela_ate), com o teto dela; o ambiente da leve não estica a janela. Início e fim vão ao log.
FILA_PESADA_LIBERACAO=1   # DV 2397: trocar para 0 quando ela cair (sem variável de ambiente, de propósito)
FILA_JANELA_LEVE_S=${REVENU_GATE_JANELA_LEVE_S:-600}             # 0: sem janela (a leve espera desde o pedido)
FILA_LOCK=${REVENU_GATE_PESADO_LOCK:-/private/tmp/revenu-gate-pesado.lock}
FILA_SENHAS=${REVENU_GATE_PESADO_SENHAS:-${FILA_LOCK%.lock}.senhas}
FILA_LOG=${REVENU_GATE_PESADO_LOG:-${FILA_LOCK%.lock}.log}   # uma linha por entrada: as leituras que abriram a vaga
FILA_VAGAS_MAX=${REVENU_GATE_VAGAS_MAX:-5}
FILA_CARGA_2=${REVENU_GATE_CARGA_2:-54}
FILA_CARGA_0=${REVENU_GATE_CARGA_0:-90}
FILA_TIMEOUT_S=${REVENU_GATE_TIMEOUT_S:-5400}
FILA_SWAP_MAX=${REVENU_GATE_SWAP_MAX:-85}
FILA_CRDB_MAX=${REVENU_GATE_CRDB_MAX:-3}
FILA_LEVE_MAX=${REVENU_GATE_LEVE_MAX:-2}
FILA_ESPERA_ALERTA_S=${REVENU_GATE_ESPERA_ALERTA_S:-1800} # --status marca ESPERA>30min (vai ao relato)
FILA_TETO_N1_S=${REVENU_GATE_TETO_N1_S:-900} FILA_TETO_N2_S=${REVENU_GATE_TETO_N2_S:-2700} # --status marca TETO>N1/N2 pelo nível declarado
FILA_LEVE_WARN=${REVENU_GATE_LEVE_WARN:-2}
FILA_ESPERA_S=${REVENU_GATE_ESPERA_S:-10}     # intervalo de quem espera vaga (os testes encurtam)
FILA_VOLTA_S=${REVENU_GATE_VOLTA_S:-5}         # intervalo do laço do modo comando
FILA_LEVE_CONFERE=${REVENU_GATE_LEVE_CONFERE:-2} # a cada quantas voltas a pista sem banco confere o que a rodada subiu
FILA_LOCK_BANCO=$FILA_LOCK
FILA_LOCK_LEVE=${FILA_LOCK%.lock}.leve.lock
fila_leve=0
fila_pesada=0                                                    # 1: rodada pesada, exclusiva (dono, 2026-11-22)
FILA_EXCLUSIVA=${REVENU_GATE_EXCLUSIVA:-1}
FILA_MEM_LIVRE_MIN_MB=${REVENU_GATE_MEM_LIVRE_MIN_MB:-6144}      # 0 desliga a conferência da memória da VM
FILA_MEM_PRAZO_S=${REVENU_GATE_MEM_PRAZO_S:-15}                  # docker travado não trava a fila: passou, «não lida»
FILA_LEVE_DURANTE_PESADA=${REVENU_GATE_LEVE_DURANTE_PESADA:-1} # leves sem contêiner junto de uma pesada (2026-11-22)
FILA_DECIDE=${FILA_LOCK%.lock}.decide # lock curto, comum às duas pistas: conferir a exclusiva e pegar a vaga sem corrida
FILA_LIMITES=${REVENU_GATE_LIMITES_ARQ:-${FILA_LOCK%.lock}.limites} # KEY=N por linha, relido a cada volta; vence o ambiente
FILA_PRIORIDADE=${REVENU_GATE_PRIORIDADE_ARQ:-${FILA_LOCK%.lock}.prioridade} # um regex de dono por linha; vazio ou ausente: sem prioridade
# Reserva (dono, select no Core e na sessão Baremetal em 2026-11-13, MS-023 1.43): com o arquivo presente, nas duas
# pistas só abre vaga a senha cujo dono casa o .prioridade; as outras esperam (o --status marca RESERVA) e não travam a
# cabeça. Quem já roda termina. .prioridade vazio com .reserva presente = nenhuma vaga nova. Sem o arquivo, nada muda.
FILA_RESERVA=${REVENU_GATE_RESERVA_ARQ:-${FILA_LOCK%.lock}.reserva}
FILA_VAGAS_WARN=${REVENU_GATE_VAGAS_WARN:-3}
# Estado da CONVENTIONS § 6.7 e § 6.10: provas por hash, falhas, DLQ e breaker. O /tmp perde arquivo: perder só faz rodar de novo.
FILA_ESTADO=${REVENU_GATE_ESTADO:-${FILA_LOCK%.lock}.estado}
FILA_DLQ=${REVENU_GATE_DLQ:-0}                   # DLQ só para quem pede, até o piloto (dono, select no Core em 2026-11-07)
FILA_DLQ_MAX=${REVENU_GATE_DLQ_MAX:-3}           # falhas seguidas do mesmo dono e comando até a DLQ
FILA_BREAKER_MAX=${REVENU_GATE_BREAKER_MAX:-3}   # falhas de ambiente seguidas, de qualquer dono, até abrir o breaker
FILA_BREAKER_S=${REVENU_GATE_BREAKER_S:-900}     # breaker aberto recusa entrada por este tempo; depois, uma tentativa decide
FILA_NUNCA='^(govulncheck.*|.*vermelh.*)$'       # passos que nunca reaproveitam prova
fila_senha=""; fila_vaga=""

fila_limites() { # relê o arquivo de limites: uma mudança vale na volta seguinte, também para quem já espera neste código
  [ -f "$FILA_LIMITES" ] || return 0
  local k v proprio=0
  [ "$FILA_LIMITES" != "${FILA_LOCK_BANCO%.lock}.limites" ] || proprio=1
  while IFS='=' read -r k v; do
    case "$v" in ''|*[!0-9]*) continue ;; esac
    case "$k" in
      VAGAS_MAX) FILA_VAGAS_MAX=$v ;; CARGA_2) FILA_CARGA_2=$v ;; CARGA_0) FILA_CARGA_0=$v ;;
      SWAP_MAX) FILA_SWAP_MAX=$v ;; VAGAS_WARN) FILA_VAGAS_WARN=$v ;; CRDB_MAX) FILA_CRDB_MAX=$v ;;
      LEVE_MAX) FILA_LEVE_MAX=$v ;; LEVE_WARN) FILA_LEVE_WARN=$v ;;
      MEM_LIVRE_MIN_MB) FILA_MEM_LIVRE_MIN_MB=$v ;; EXCLUSIVA) FILA_EXCLUSIVA=$v ;;
      JANELA_LEVE_S) FILA_JANELA_LEVE_S=$v ;;
      # só o arquivo da própria fila: REVENU_GATE_LIMITES_ARQ, que é ambiente de quem pede, não desliga a liberação
      PESADA_LIBERACAO) [ "$proprio" != 1 ] || FILA_PESADA_LIBERACAO=$v ;;
    esac
  done < "$FILA_LIMITES"
  if [ "$fila_leve" = 1 ]; then FILA_VAGAS_MAX=$FILA_LEVE_MAX; fi
  return 0
}

fila_viva() { # <dir da senha ou do lock>: 0 se o dono está vivo
  local p; p=$(cat "$1/pid" 2>/dev/null || true)
  if [ -z "$p" ]; then
    local m; m=$(stat -f %m "$1" 2>/dev/null || stat -c %Y "$1" 2>/dev/null) || return 1
    [ $(( $(date +%s) - m )) -lt 60 ]; return
  fi
  kill -0 "$p" 2>/dev/null
}

fila_carga() { # carga de 5 min, inteira (o macOS em pt_BR escreve 30,09); a de 1 min oscila 90↔140 e abria
  # vaga numa queda curta (decisão do dono, 05/11)
  local c; c=$(sysctl -n vm.loadavg 2>/dev/null | awk '{print $3}') || true
  [ -n "$c" ] || c=$(awk '{print $2}' /proc/loadavg 2>/dev/null)
  c=${c%%[.,]*}; echo "${c:-0}"
}

fila_swap_pct() { # swap usado, em % inteiro (0 sem swap ou fora do macOS)
  sysctl -n vm.swapusage 2>/dev/null | awk '{gsub(",",".");t=$3+0;u=$6+0; print (t>0 ? int(u*100/t) : 0)}' || echo 0
}

fila_pressao() { # nível de pressão de memória do kernel: 1 normal, 2 warn, 4 critical (1 fora do macOS)
  sysctl -n kern.memorystatus_vm_pressure_level 2>/dev/null || echo 1
}

fila_rodando() { # quantas vagas estão ocupadas por um dono vivo
  local d n=0; for d in "$FILA_LOCK" "$FILA_LOCK".[0-9]*; do [ -d "$d" ] && fila_viva "$d" && n=$((n + 1)); done; echo "$n"
}

fila_crdb_vivos() { # CockroachDB de gate vivos: contêiner (label revenu.gate.dono, imagem cockroach) + binário nativo no host
  local c=0 n; n=$(pgrep -f '^[^ ]*cockroach start' 2>/dev/null | wc -l | tr -d ' ') # a malha roda na VM do Docker: não aparece aqui
  # com prazo (2026-11-22): docker travado travava a cabeça aqui; sem resposta, conta 0, como na falha
  command -v docker >/dev/null 2>&1 && c=$(fila_com_prazo "$FILA_MEM_PRAZO_S" docker ps --filter label=revenu.gate.dono --format '{{.Image}}' | grep -c -i cockroach || true)
  echo $(( ${c:-0} + ${n:-0} ))
}

fila_prioritario() { # <dono>: 0 se a senha passa à frente (REVENU_GATE_PRIORIDADE=1, ou dono casa o regex do arquivo)
  [ "${REVENU_GATE_PRIORIDADE:-0}" = 1 ] && return 0
  [ -s "$FILA_PRIORIDADE" ] && printf '%s\n' "$1" | grep -E -q -f "$FILA_PRIORIDADE"
}

fila_barrada() { # <dir da senha>: 0 se a reserva está ligada e o dono da senha não casa o .prioridade
  [ -e "$FILA_RESERVA" ] || return 1
  ! { [ -s "$FILA_PRIORIDADE" ] && grep -E -q -f "$FILA_PRIORIDADE" "$1/dono" 2>/dev/null; }
}

fila_nova_senha() { # <dono>: número da próxima senha. Prioritária: faixa negativa, em ordem de chegada, à frente das
  # positivas; quem já espera lê a pasta por `sort -n` e passa a esperar atrás dela sem mudar nada
  local nums; nums=$(find "$FILA_SENHAS" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort -n)
  if fila_prioritario "$1"; then
    local u; u=$(printf '%s\n' "$nums" | grep '^-' | tail -1); echo $(( ${u:--1000000} + 1 ))
  else
    local u; u=$(printf '%s\n' "$nums" | grep -v '^-' | tail -1); echo $(( ${u:-0} + 1 ))
  fi
}

fila_rodando_tudo() { # gates rodando nas duas pistas, com banco e sem banco
  local d n=0
  for d in "$FILA_LOCK_BANCO" "$FILA_LOCK_BANCO".[0-9]* "$FILA_LOCK_LEVE" "$FILA_LOCK_LEVE".[0-9]*; do
    [ -d "$d" ] && fila_viva "$d" && n=$((n + 1))
  done
  echo "$n"
}

fila_com_prazo() { # <segundos> <comando…>: a saída do comando; rc 124 se passar do prazo (o macOS não tem `timeout`)
  local s=$1 f q i rc=0; shift; f=$(mktemp "${TMPDIR:-/tmp}/fila-prazo.XXXXXX") || return 1
  "$@" > "$f" 2>/dev/null & q=$!
  for ((i = 0; i < s * 4; i++)); do kill -0 "$q" 2>/dev/null || break; sleep 0.25; done
  if kill -0 "$q" 2>/dev/null; then kill "$q" 2>/dev/null; wait "$q" 2>/dev/null; rm -f "$f"; return 124; fi
  wait "$q" || rc=$?; cat "$f"; rm -f "$f"; return "$rc"
}

fila_mem_livre_mb() { # memória livre da VM do Docker, em MiB inteiros; «-» se não deu para ler (dono, 2026-11-22)
  # MemTotal do `docker info` (bytes) menos a soma do uso dos contêineres no `docker stats` («1.5GiB / 15.6GiB»). Não vê
  # o que é da VM e não de contêiner (kernel, dockerd, buildkit): a sobra real é um pouco menor que a lida.
  command -v docker >/dev/null 2>&1 || { echo -; return 0; }
  local total uso
  total=$(fila_com_prazo "$FILA_MEM_PRAZO_S" docker info --format '{{.MemTotal}}') || { echo -; return 0; }
  case "$total" in ''|*[!0-9]*|0) echo -; return 0 ;; esac
  uso=$(fila_com_prazo "$FILA_MEM_PRAZO_S" docker stats --no-stream --format '{{.MemUsage}}') || { echo -; return 0; }
  printf '%s\n' "$uso" | LC_ALL=C awk -v t="$total" '
    BEGIN { m["B"] = 1; m["kB"] = 1e3; m["KB"] = 1e3; m["MB"] = 1e6; m["GB"] = 1e9; m["TB"] = 1e12
            m["KiB"] = 1024; m["MiB"] = 1048576; m["GiB"] = 1073741824; m["TiB"] = 1099511627776 }
    { s = $0; sub(/\/.*/, "", s); gsub(/[ \t]/, "", s) }
    s == "" || s ~ /^-+$/ { next }            # contêiner subindo: o stats ainda não tem número
    !match(s, /^[0-9.]+/) { ruim = 1; next }
    { n = substr(s, 1, RLENGTH); u = substr(s, RLENGTH + 1); if (!(u in m)) { ruim = 1; next }; usado += n * m[u] }
    END { if (ruim) { print "-"; exit }; l = (t - usado) / 1048576; printf "%d\n", (l < 0 ? 0 : l) }'
}

fila_gb() { # <MiB>: em GB com vírgula, como o dono lê («6», «5,5»)
  LC_ALL=C awk -v m="$1" 'BEGIN { printf "%.1f\n", m / 1024 }' | sed 's/\.0$//; s/\./,/'
}

fila_pesadas_vivas() { # rodadas pesadas com vaga e dono vivo: toda vaga da pista banco e a da leve marcada `pesada`
  local d n=0
  for d in "$FILA_LOCK_BANCO" "$FILA_LOCK_BANCO".[0-9]*; do [ -d "$d" ] && fila_viva "$d" && n=$((n + 1)); done
  for d in "$FILA_LOCK_LEVE" "$FILA_LOCK_LEVE".[0-9]*; do [ -f "$d/pesada" ] && fila_viva "$d" && n=$((n + 1)); done
  echo "$n"
}

fila_pesada_esperando() { # 0 se uma senha pesada viva espera só pela exclusiva: as leves novas esperam com ela, senão
  # a pista leve, sempre com alguém rodando, nunca deixaria a pesada entrar. fila_janela_ate fica com o fim da janela
  # da leve (dono, 2026-11-22): o menor entre as pesadas que esperam; 0 se alguma não abriu janela
  local s a r=1; fila_janela_ate=""
  for s in "${FILA_LOCK_BANCO%.lock}".senhas/* "${FILA_LOCK_LEVE%.lock}".senhas/*; do
    if ! { [ -f "$s/pesada" ] && [ ! -f "$s/vaga" ] && [ "$(cat "$s/motivo" 2>/dev/null)" = exclusiva ]; }; then continue; fi
    fila_viva "$s" || continue
    a=$(cat "$s/janela_ate" 2>/dev/null); case "$a" in ''|*[!0-9]*) a=0 ;; esac
    if [ -z "$fila_janela_ate" ] || [ "$a" -lt "$fila_janela_ate" ]; then fila_janela_ate=$a; fi
    r=0
  done
  return "$r"
}
fila_janela_ate=""

fila_janela_aberta() { # 0 se uma pesada espera a exclusiva e a janela da leve sem contêiner ainda não passou do teto
  fila_pesada_esperando && [ "$(date +%s)" -lt "${fila_janela_ate:-0}" ]
}

fila_janela_pesada() { # <senha> <motivo>: a pesada que espera só pela exclusiva abre a janela da leve, e fecha no teto
  local d=$FILA_SENHAS/$1 agora a
  [ "$2" = exclusiva ] || return 0
  agora=$(date +%s)
  if [ ! -f "$d/janela_ate" ]; then
    # antes do `motivo`: quem lê a pesada como «na fila» já acha o fim da janela
    echo $((agora + FILA_JANELA_LEVE_S)) > "$d/janela_ate"
    echo "$(date -u +%FT%TZ) janela-leve inicio senha=$1 dono=$(cat "$d/dono" 2>/dev/null) teto=${FILA_JANELA_LEVE_S}s" \
      >> "$FILA_LOG" 2>/dev/null || true
  elif [ ! -f "$d/janela_fim" ]; then
    a=$(cat "$d/janela_ate" 2>/dev/null); case "$a" in ''|*[!0-9]*) a=0 ;; esac
    [ "$agora" -ge "$a" ] && fila_janela_fecha "$1" 'teto: a fila drena para a pesada'
  fi
  return 0
}

fila_janela_fecha() { # <senha> <motivo>: registra o fim da janela da leve, uma vez só
  local d=$FILA_SENHAS/$1
  [ -f "$d/janela_ate" ] && [ ! -f "$d/janela_fim" ] || return 0
  : > "$d/janela_fim"
  echo "$(date -u +%FT%TZ) janela-leve fim senha=$1 dono=$(cat "$d/dono" 2>/dev/null) motivo=«$2»" >> "$FILA_LOG" 2>/dev/null || true
}

fila_pede_pesada() { # <dono>: registra a liberação no log, ou recusa a pesada antes da senha e encerra (rc 77)
  local m; m=$(printf '%s' "${REVENU_GATE_PESADA_LIBERADA:-}" | tr '\n\r\t' '   ' | cut -c1-200)
  local pista=banco; [ "$fila_leve" = 0 ] || pista=leve
  if fila_liberada; then
    [ "$FILA_PESADA_LIBERACAO" = 1 ] || return 0
    echo "$(date -u +%FT%TZ) pesada-liberada dono=$1 pid=$$ pista=$pista motivo=«${m}»" >> "$FILA_LOG" 2>/dev/null || true
    return 0
  fi
  echo "$(date -u +%FT%TZ) pesada-recusada dono=$1 pid=$$ pista=$pista motivo=«sem REVENU_GATE_PESADA_LIBERADA»" \
    >> "$FILA_LOG" 2>/dev/null || true
  {
    echo "fila do gate pesado: rodada pesada de $1 (pista $pista) recusada, sem senha (rc 77)."
    echo "  A DV 2397 diz que hoje vale só a N1 unitária: pesada só com ordem do dono."
    echo "  Com a ordem, peça de novo com o motivo: REVENU_GATE_PESADA_LIBERADA='<ordem do dono>' <o mesmo comando>"
  } >&2
  exit 77
}

fila_liberada() { # 0 se a pesada pode pedir senha: regra desligada ou REVENU_GATE_PESADA_LIBERADA com motivo (DV 2397)
  local m=${REVENU_GATE_PESADA_LIBERADA:-}
  [ "$FILA_PESADA_LIBERACAO" = 1 ] || return 0
  [ -n "${m//[[:space:]]/}" ]
}

fila_exclusiva() { # o motivo se a regra da pesada exclusiva barra a entrada agora; nada se não barra
  [ "$FILA_EXCLUSIVA" = 1 ] || return 0
  if [ "$fila_pesada" = 1 ]; then
    [ "$(fila_rodando_tudo)" -eq 0 ] || echo exclusiva
  elif [ "$(fila_pesadas_vivas)" -gt 0 ]; then
    # durante a pesada, a leve sem contêiner (sem REVENU_GATE_DOCKER) abre até FILA_LEVE_DURANTE_PESADA vaga (dono,
    # 2026-11-22: às 15:13 BRT a leve ficou 46 min parada atrás do mutation-gate do accountos); a memória ela confere
    # em fila_avaliar. Com contêiner, segue exclusiva
    if [ "$fila_leve" = 1 ] && [ "${REVENU_GATE_DOCKER:-0}" != 1 ]; then
      [ "$(fila_rodando)" -lt "$FILA_LEVE_DURANTE_PESADA" ] && return 0
      echo 'exclusiva (já roda leve sem contêiner durante a pesada)'; return 0
    fi
    echo exclusiva
  elif fila_pesada_esperando; then
    # janela da leve (dono, 2026-11-22): a sem contêiner entra como sempre até o teto contado do pedido da exclusiva
    if [ "$fila_leve" = 1 ] && [ "${REVENU_GATE_DOCKER:-0}" != 1 ] && [ "$(date +%s)" -lt "${fila_janela_ate:-0}" ]; then
      return 0
    fi
    echo 'exclusiva (pesada na fila)'
  fi
  return 0
}

fila_cede() { # <dir da senha> <minha senha>: 0 se ela é leve com contêiner, espera a exclusiva e eu sou leve sem contêiner
  [ "$fila_leve" = 1 ] && [ "$fila_pesada" != 1 ] && [ "${REVENU_GATE_DOCKER:-0}" != 1 ] || return 1
  [ "${1##*/}" != "$2" ] && [ -f "$1/docker" ] && [ ! -f "$1/pesada" ] || return 1
  case "$(cat "$1/motivo" 2>/dev/null)" in *exclusiva*) return 0 ;; esac
  return 1
}

fila_decide_pega() { # 0 com o lock de decisão; 1 se outro decide agora (esta volta passa). Lock de dono morto é retomado
  if mkdir "$FILA_DECIDE" 2>/dev/null; then echo "$$" > "$FILA_DECIDE/pid"; return 0; fi
  fila_viva "$FILA_DECIDE" && return 1
  rm -rf "$FILA_DECIDE"; mkdir "$FILA_DECIDE" 2>/dev/null || return 1
  echo "$$" > "$FILA_DECIDE/pid"
}

fila_decide_solta() {
  if [ "$(cat "$FILA_DECIDE/pid" 2>/dev/null)" = "$$" ]; then rm -rf "$FILA_DECIDE"; fi
  return 0
}

fila_vagas_leve() { # a pista sem banco: não conta CockroachDB; carga, swap e pressão continuam valendo
  local c p; c=$(fila_carga); p=$(fila_pressao)
  [ "${REVENU_GATE_RODADA:-0}" = 1 ] && c=0 # a rodada não corta por carga, como em fila_vagas_de
  [ "$c" -ge "$FILA_CARGA_0" ] && { echo 0; return; }
  if [ "$p" -ge 4 ]; then # critical: nenhuma vaga nova, salvo com a máquina sem gate nenhum
    if [ "$(fila_rodando_tudo)" -eq 0 ]; then echo 1; else echo 0; fi; return
  fi
  if [ "$p" -ge 2 ] && [ "$(fila_swap_pct)" -gt "$FILA_SWAP_MAX" ]; then
    if [ "$(fila_rodando)" -lt "$FILA_LEVE_WARN" ]; then echo "$FILA_LEVE_MAX"; else echo 0; fi; return
  fi
  if [ "$c" -lt "$FILA_CARGA_2" ] || [ "$FILA_LEVE_MAX" -le 1 ]; then echo "$FILA_LEVE_MAX"; return; fi
  local passo=$(( (FILA_CARGA_0 - FILA_CARGA_2) / (FILA_LEVE_MAX - 1) )) v
  v=$(( FILA_LEVE_MAX - 1 - (c - FILA_CARGA_2) / (passo > 0 ? passo : 1) ))
  [ "$v" -lt 1 ] && v=1; echo "$v"
}

fila_confere_leve() { # <dono> <pid> <início>: na pista sem banco, o motivo se a rodada subiu banco; nada se não
  # A pista vale pela declaração (REVENU_GATE_SEM_BANCO=1), não pelo nome do comando (dono, select na sessão Baremetal em
  # 2026-11-08: um e2e em SQLite caía na pista com banco por ter «e2e» no nome). Durante a rodada, a fila confere o que
  # o comando subiu: (1) processo descendente que sobe banco ou contêiner; (2) contêiner CockroachDB ou Kafka criado
  # depois do início, com o label revenu.gate.dono do dono, desde que nenhum gate com banco esteja vivo. Quem sobe banco
  # declarado leve é encerrado. Contêiner SEM o label não conta (dono, 2026-11-22): às 11:46 um `docker run` solto da
  # bancada da KeycloakOS derrubou duas leves da PaymentOS (rc 66), e um redpanda de testcontainers de outra sessão
  # derrubou a N1 da AuthorityOS duas vezes. O da própria rodada sem label segue pego pelos descendentes (1).
  [ "$fila_leve" = 1 ] || return 0
  local dono=$1 pid=$2 inicio=$3 p cmd
  for p in $(fila_descendentes "$pid"); do
    cmd=$(ps -o command= -p "$p" 2>/dev/null) || continue
    case "$cmd" in
      *"cockroach start"*|*kafka-server-start*) echo "processo $p: ${cmd:0:120}"; return 0 ;;
      # docker só conta quando sobe banco: o act, o hadolint e o dockle rodam `docker run` sem banco (MS-022 1.26).
      # Casa a IMAGEM, não a linha inteira (2026-11-23, IdentityOS): `-v …/wt-kafka:/src` do lint e um `bash … docker run
      # … internal/kafkaauth` (o processo é o bash, não o docker) derrubavam leves sem banco com rc 66.
      *docker*)
        case "$(fila_imagem_docker "$cmd" | tr '[:upper:]' '[:lower:]')" in
          *cockroach*|*kafka*|*redpanda*) echo "processo $p: ${cmd:0:120}"; return 0 ;;
        esac ;;
    esac
  done
  command -v docker >/dev/null 2>&1 || return 0
  [ "$(FILA_LOCK=$FILA_LOCK_BANCO fila_rodando)" = 0 ] || return 0
  local id img rot criado ts d; d=$(printf '%s' "$dono" | tr '[:upper:]' '[:lower:]')
  while IFS='|' read -r id img rot criado; do
    [ -n "$id" ] || continue
    case "$(printf '%s' "$img" | tr '[:upper:]' '[:lower:]')" in *cockroach*|*kafka*|*redpanda*) ;; *) continue ;; esac
    case "$(printf '%s' "$rot" | tr '[:upper:]' '[:lower:]')" in "$d") ;; *) continue ;; esac
    ts=$(date -j -f '%Y-%m-%d %H:%M:%S %z' "${criado:0:25}" +%s 2>/dev/null || date -d "${criado:0:25}" +%s 2>/dev/null) || continue # «2026-11-08 19:15:38 +0100 WAT»
    [ "$ts" -ge "$inicio" ] && { echo "contêiner $id ($img) criado durante a rodada"; return 0; }
  done < <(docker ps --format '{{.ID}}|{{.Image}}|{{.Label "revenu.gate.dono"}}|{{.CreatedAt}}' 2>/dev/null || true)
  return 0
}

fila_imagem_docker() { # <linha de comando>: a imagem de um `docker run|create` (ou o resto da linha de um compose); vazio se o
  # processo não é o próprio docker. Opções com valor separado pulam o valor (-v x, --name y); `--opt=valor` é um token só.
  local -a a; read -r -a a <<<"$1"; local i=1 n=${#a[@]} t
  [ "$n" -gt 0 ] || return 0
  case "${a[0]##*/}" in
    docker-compose) printf '%s' "${a[*]:1}"; return 0 ;;
    docker|com.docker.cli) ;;
    *) return 0 ;;
  esac
  while [ "$i" -lt "$n" ]; do # opções globais (--context x, -H y) até o subcomando
    case "${a[$i]}" in
      --context|-c|-H|--host|--config|-l|--log-level) i=$((i + 2)) ;;
      -*) i=$((i + 1)) ;;
      *) break ;;
    esac
  done
  [ "$i" -lt "$n" ] || return 0
  case "${a[$i]}" in
    compose) printf '%s' "${a[*]:$((i + 1))}"; return 0 ;;
    container) i=$((i + 1)); case "${a[$i]:-}" in run|create) ;; *) return 0 ;; esac ;;
    run|create) ;;
    *) return 0 ;;
  esac
  i=$((i + 1))
  while [ "$i" -lt "$n" ]; do
    t=${a[$i]}
    case "$t" in
      --*=*) i=$((i + 1)) ;;
      -v|--volume|-e|--env|--env-file|--name|--network|--net|-w|--workdir|--entrypoint|-p|--publish|-l|--label|\
      --label-file|-u|--user|--mount|-m|--memory|--memory-swap|--cpus|-c|--cpu-shares|--cpuset-cpus|--platform|-h|\
      --hostname|--add-host|--cap-add|--cap-drop|--device|--dns|--log-driver|--log-opt|--restart|--security-opt|--tmpfs|\
      --ulimit|--pull|--cidfile|--stop-signal|--stop-timeout|--ipc|--pid|--shm-size|--gpus|--runtime|--network-alias|\
      --link|--volumes-from|--expose|--group-add|--userns|--uts|-a|--attach|--health-cmd|--health-interval|--annotation)
        i=$((i + 2)) ;;
      -*) i=$((i + 1)) ;;
      *) printf '%s' "$t"; return 0 ;;
    esac
  done
}

fila_descendentes() { # <pid>: os descendentes, um por linha
  local f; for f in $(pgrep -P "$1" 2>/dev/null); do echo "$f"; fila_descendentes "$f"; done
}

fila_avaliar() { # «vagas carga swap% pressão rodando crdb memvm motivo…»: as vagas e as leituras que as decidiram, numa
  # linha. memvm: MiB livres na VM do Docker, «-» não lida, «desligada» com o mínimo 0; motivo: «-» ou por que zerou
  local c p sw r k v m x mot=""; c=$(fila_carga); p=$(fila_pressao); sw=$(fila_swap_pct); r=$(fila_rodando); k=$(fila_crdb_vivos)
  if [ "$fila_leve" = 1 ]; then v=$(fila_vagas_leve); else v=$(fila_vagas_de "$c" "$p" "$sw" "$r" "$k"); fi
  # na pista leve só espera memória quem declara contêiner, REVENU_GATE_DOCKER=1 (dono, 2026-11-22, defeito achado pela
  # AccountOS: o mínimo travava o `go test -short`, que roda no macOS e não usa a VM); banco e pesada esperam sempre
  m=desligada
  if [ "$fila_leve" = 1 ] && [ "$fila_pesada" != 1 ] && [ "${REVENU_GATE_DOCKER:-0}" != 1 ]; then m=dispensada
    # durante a pesada ela divide a máquina: lê a memória e, sem leitura, não entra (dono, 2026-11-22); na janela da
    # pesada que espera a exclusiva também, porque a pesada vai precisar da VM (mesmo dia, «com a memória mínima»)
    if [ "$FILA_EXCLUSIVA" = 1 ] && [ "$FILA_MEM_LIVRE_MIN_MB" -gt 0 ] \
      && { [ "$(fila_pesadas_vivas)" -gt 0 ] || fila_janela_aberta; }; then
      m=$(fila_mem_livre_mb)
      case "$m" in ''|*[!0-9]*) v=0; mot="memória da VM não lida (leve sem contêiner junto da pesada)" ;; esac
    fi
  elif [ "$FILA_MEM_LIVRE_MIN_MB" -gt 0 ]; then m=$(fila_mem_livre_mb); fi
  case "$m" in
    ''|*[!0-9]*) ;;   # não lida (ou desligada): não bloqueia
    *) [ "$m" -ge "$FILA_MEM_LIVRE_MIN_MB" ] || { v=0; mot="memória < $(fila_gb "$FILA_MEM_LIVRE_MIN_MB") GB"; } ;;
  esac
  x=$(fila_exclusiva); [ -z "$x" ] || { v=0; mot=${mot:+$mot; }$x; }
  echo "$v $c $sw $p $r $k $m ${mot:--}"
}

fila_vagas() { # quantos gates podem rodar juntos agora, nesta pista (teto, não índice de vaga)
  fila_avaliar | cut -d' ' -f1
}

fila_vagas_de() { # <carga> <pressão> <swap%> <rodando> <crdb>: a regra, sem ler nada (as leituras vêm de fila_avaliar)
  local c=$1 p=$2 sw=$3 r=$4 k=${5:-0}
  [ "${REVENU_GATE_RODADA:-0}" = 1 ] && c=0 # rodada: corta por memória e pelo teto de CRDB, não por carga (CONVENTIONS § 6.10)
  [ "$c" -ge "$FILA_CARGA_0" ] && { echo 0; return; }
  # um CockroachDB de gate por vez: com um vivo, nenhuma vaga nova abre (decisão do dono, 05/11)
  [ "$k" -ge "$FILA_CRDB_MAX" ] && { echo 0; return; }
  if [ "$p" -ge 2 ] && [ "$sw" -gt "$FILA_SWAP_MAX" ]; then
    # memória apertada: o teto é de gates RODANDO (2 em warn, 1 em critical), não de índice de vaga; abaixo
    # do teto, qualquer vaga livre serve, e no teto nenhuma abre
    local teto=$FILA_VAGAS_WARN; [ "$p" -ge 4 ] && teto=1
    if [ "$r" -lt "$teto" ]; then echo "$FILA_VAGAS_MAX"; else echo 0; fi; return
  fi
  # entre CARGA_2 e CARGA_0 as vagas caem uma a uma, em degraus iguais
  if [ "$c" -ge "$FILA_CARGA_0" ]; then echo 0
  elif [ "$c" -lt "$FILA_CARGA_2" ] || [ "$FILA_VAGAS_MAX" -le 1 ]; then echo "$FILA_VAGAS_MAX"
  else
    local passo=$(( (FILA_CARGA_0 - FILA_CARGA_2) / (FILA_VAGAS_MAX - 1) )) v
    v=$(( FILA_VAGAS_MAX - 1 - (c - FILA_CARGA_2) / (passo > 0 ? passo : 1) ))
    [ "$v" -lt 1 ] && v=1; echo "$v"
  fi
}

fila_progresso_resumo() { # <progresso.tsv>: «passo pacote desde · feitos/total falhas» do que está em curso, ou «-»
  [ -r "$1" ] || { echo "- (progresso ilegível)"; return; }
  awk -F'\t' '
    /^#resumo\t/ { r = $3 " " $4; next }
    /^#erro-conf\t/ { ec = ec (ec == "" ? "" : ",") $2; next }   # rc 2 do rodada-escopo.sh: passos.conf inválido
    /^#/ || NR == 1 && $1 == "ts_utc" { next }
    NF >= 4 { k = $2 SUBSEP $3
      if ($4 == "inicio") { ini[k] = $1; ordem[++n] = k } else { delete ini[k]; fim++; if ($4 == "falha") f++; if ($4 == "estourou") e++ } }
    END {
      for (i = n; i >= 1; i--) if (ordem[i] in ini) { split(ordem[i], a, SUBSEP); c = a[1] " " a[2] " desde " ini[ordem[i]]; break }
      if (c == "") c = "-"
      if (r == "") r = "terminados=" fim + 0 " falhas=" f + 0
      if (e > 0) r = r " estourou=" e " ESTOUROU" # passo cortado pelo teto: ⚠️, não vermelho (CONVENTIONS § 6.6)
      if (ec != "") r = r " ERRO-CONF erro de passos.conf (" ec ")"
      print c " · " r
    }' "$1"
}

fila_status() { # uma linha por senha viva das duas pistas: pista, senha, dono, vaga ou espera, e o progresso declarado
  local pista dir s pid estado prog agora desde
  for pista in banco leve; do
    dir=${FILA_LOCK_BANCO%.lock}.senhas; [ "$pista" = leve ] && dir=${FILA_LOCK_LEVE%.lock}.senhas
    [ -d "$dir" ] || continue
    for s in $(find "$dir" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort -n); do
      fila_viva "$dir/$s" || continue
      pid=$(cat "$dir/$s/pid" 2>/dev/null || echo -)
      # tempo: na vaga, desde que entrou; na espera, desde a senha. Espera acima de 30 min vai ao relato (dono, 2026-11-08)
      agora=$(date +%s)
      if [ -f "$dir/$s/vaga" ]; then
        desde=$(stat -f %m "$dir/$s/vaga" 2>/dev/null || stat -c %Y "$dir/$s/vaga"); estado="vaga ${pista}:$(basename "$(cat "$dir/$s/vaga")") há $(( (agora - desde) / 60 )) min"
        [ ! -s "$dir/$s/durante" ] || estado="$estado ($(cat "$dir/$s/durante"))"
        # teto de parede do nível declarado (N1 15 min, N2 45 min; CONVENTIONS § 6.10): passou, vira investigação com dono
        case $(cat "$dir/$s/nivel" 2>/dev/null) in
          N1) [ $(( agora - desde )) -gt "$FILA_TETO_N1_S" ] && estado="$estado TETO>N1" ;;
          N2) [ $(( agora - desde )) -gt "$FILA_TETO_N2_S" ] && estado="$estado TETO>N2" ;;
        esac
      else
        desde=$(stat -f %m "$dir/$s/pid" 2>/dev/null || stat -c %Y "$dir/$s/pid" 2>/dev/null || echo "$agora"); estado="espera há $(( (agora - desde) / 60 )) min"
        [ $(( agora - desde )) -gt "$FILA_ESPERA_ALERTA_S" ] && estado="$estado ESPERA>30min"
        fila_barrada "$dir/$s" && estado="$estado RESERVA"
        # por que a cabeça não entrou na última volta: «exclusiva», «memória < 6 GB» (dono, 2026-11-22)
        [ ! -s "$dir/$s/motivo" ] || estado="$estado ($(cat "$dir/$s/motivo"))"
      fi
      [ ! -f "$dir/$s/pesada" ] || estado="$estado pesada"
      [ "$(cat "$dir/$s/nivel_teste" 2>/dev/null)" != unitaria ] || estado="$estado nível=unitária"
      prog=-; [ -f "$dir/$s/progresso" ] && prog=$(fila_progresso_resumo "$(cat "$dir/$s/progresso")")
      printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$pista" "$s" "$(cat "$dir/$s/dono" 2>/dev/null)" "$pid" "$estado" "$prog"
    done
  done
  fila_status_memoria
}

fila_status_memoria() { # a última linha do --status, com «#» na frente: a memória livre da VM do Docker lida agora
  fila_limites
  local m min; min=$(fila_gb "$FILA_MEM_LIVRE_MIN_MB")
  if [ "$FILA_MEM_LIVRE_MIN_MB" -le 0 ]; then echo "# memória da VM: conferência desligada (MEM_LIVRE_MIN_MB=0)"; return 0; fi
  m=$(fila_mem_livre_mb)
  case "$m" in
    ''|*[!0-9]*) echo "# memória da VM: não lida (sem docker ou leitura falha; não bloqueia; mínimo $min GB)" ;;
    *) if [ "$m" -ge "$FILA_MEM_LIVRE_MIN_MB" ]; then echo "# memória da VM: $(fila_gb "$m") GB livres (mínimo $min GB)"
       else echo "# memória da VM: $(fila_gb "$m") GB livres, memória < $min GB: nenhuma rodada nova"; fi ;;
  esac
}

fila_ancestral() { # <pid>: 0 se o pid é este processo ou um ancestral dele
  local p=$$ i
  for ((i = 0; i < 64; i++)); do
    [ "$p" = "$1" ] && return 0
    case "$p" in ''|0|1|*[!0-9]*) return 1 ;; esac
    p=$(ps -o ppid= -p "$p" 2>/dev/null | tr -d ' ')
  done
  return 1
}

fila_reentrada() { # <dono>: 0 se a chamada é aninhada na senha de um ancestral vivo (REVENU_GATE_SENHA=<pista>:<n>:<pid>).
  # Dono, 2026-11-22: às 13:34 BRT a fila travou. A pesada «exchangeos/develop-ef84bda/runsh» (senha 1) rodava o run.sh, que
  # chamou a fila de novo e esperou na leve (senha 9) atrás da cabeça 6 da KeycloakOS, que esperava a pesada pela
  # exclusiva: uma esperava a outra. Aninhada não pede senha; senha herdada de pid morto ou que não é ancestral (env vazada)
  # segue o caminho normal. As duas decisões vão ao log.
  local h=${REVENU_GATE_SENHA:-} pista n pid motivo="" dir
  [ -n "$h" ] || return 1
  IFS=: read -r pista n pid <<< "$h"
  dir=${FILA_LOCK_BANCO%.lock}.senhas; [ "$pista" = leve ] && dir=${FILA_LOCK_LEVE%.lock}.senhas
  case "$pid" in
    ''|*[!0-9]*) motivo="senha herdada ilegível" ;;
    *) if ! kill -0 "$pid" 2>/dev/null; then motivo="senha herdada de pid morto"
       elif ! fila_ancestral "$pid"; then motivo="senha herdada de pid que não é ancestral (env vazada)"
       # a senha tem de existir nesta fila, com o mesmo pid: env de outra fila (a de um teste, no /tmp) não vale
       elif [ "$(cat "$dir/$n/pid" 2>/dev/null)" != "$pid" ]; then motivo="senha herdada não existe nesta fila"; fi ;;
  esac
  if [ -n "$motivo" ]; then
    echo "$(date -u +%FT%TZ) nao-reentrante dono=$1 pid=$$ herdada=$h motivo=«${motivo}»" >> "$FILA_LOG" 2>/dev/null || true
    return 1
  fi
  echo "$(date -u +%FT%TZ) reentrante dono=$1 pid=$$ pista=$pista senha=$n pid_herdado=$pid" >> "$FILA_LOG" 2>/dev/null || true
  echo "fila do gate pesado: $1 roda dentro da senha $pista:$n (pid $pid), sem senha nova (reentrante)" >&2
}

fila_reentrante=0; fila_reentrada_vista=0

fila_entrar() { # <dono>
  # chamada aninhada (o run.sh dentro de uma rodada da fila, por exemplo): roda na vaga do ancestral (dono, 2026-11-22)
  if [ "$fila_reentrada_vista" != 1 ] && fila_reentrada "$1"; then fila_reentrante=1; return 0; fi
  local nt=${REVENU_GATE_NIVEL_TESTE:-${RODADA_NIVEL:-}}   # nível unitário: pista sem banco (dono, 2026-11-18)
  if [ "${REVENU_GATE_SEM_BANCO:-0}" = 1 ] || [ "$nt" = unitaria ]; then # pista própria: outra pasta de senhas e de vagas, sem esperar a cabeça com banco
    fila_leve=1; FILA_LOCK=$FILA_LOCK_LEVE; FILA_SENHAS=${FILA_LOCK%.lock}.senhas; FILA_VAGAS_MAX=$FILA_LEVE_MAX
  fi
  # pesada (dono, 2026-11-22): a pista banco sempre; a leve quando declara REVENU_GATE_PESADA=1 (mutation-gate/gremlins)
  fila_pesada=0; if [ "$fila_leve" = 0 ] || [ "${REVENU_GATE_PESADA:-0}" = 1 ]; then fila_pesada=1; fi
  if [ "$fila_pesada" = 1 ]; then fila_limites; fila_pede_pesada "$1"; fi   # sem liberação, sai aqui (rc 77)
  local dono=$1 n s pos cabeca vagas avisado=false leitura="" linha lv lc lsw lp lr lk lm lmot vi
  mkdir -p "$FILA_SENHAS"
  while :; do
    n=$(fila_nova_senha "$dono")
    mkdir "$FILA_SENHAS/$n" 2>/dev/null && break
  done
  fila_senha=$n; echo "$$" > "$FILA_SENHAS/$n/pid"; echo "$dono" > "$FILA_SENHAS/$n/dono"; : > "$FILA_SENHAS/$n/cabeca"
  if [ "$fila_pesada" = 1 ]; then : > "$FILA_SENHAS/$n/pesada"; fi
  if [ "${REVENU_GATE_DOCKER:-0}" = 1 ]; then : > "$FILA_SENHAS/$n/docker"; fi   # a leve sem contêiner passa dela (abaixo)
  # progresso.tsv da prova (CONVENTIONS § 6.6): o caminho fica na senha, e o --status lê dali o passo e o pacote em curso
  case ${REVENU_GATE_NIVEL_PROVA:-} in N1|N2) echo "$REVENU_GATE_NIVEL_PROVA" > "$FILA_SENHAS/$n/nivel" ;; esac
  [ "$nt" != unitaria ] || echo "$nt" > "$FILA_SENHAS/$n/nivel_teste"   # campo à parte: quem lê o `nivel` não muda
  if [ -n "${REVENU_GATE_PROGRESSO:-}" ]; then
    case $REVENU_GATE_PROGRESSO in /*) echo "$REVENU_GATE_PROGRESSO" ;; *) echo "$PWD/$REVENU_GATE_PROGRESSO" ;; esac > "$FILA_SENHAS/$n/progresso"
  fi
  while :; do
    cabeca=""; pos=0; antigos=0
    for s in $(find "$FILA_SENHAS" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; | sort -n); do
      fila_viva "$FILA_SENHAS/$s" || { rm -rf "${FILA_SENHAS:?}/$s"; continue; }
      [ -f "$FILA_SENHAS/$s/vaga" ] && continue        # já roda: não conta na fila de espera
      # Senha de código anterior (sem o marcador `cabeca`) segue a regra dela e não trava a cabeça daqui.
      if [ ! -f "$FILA_SENHAS/$s/cabeca" ] && [ "$s" != "$n" ]; then antigos=$((antigos + 1)); continue; fi
      if fila_barrada "$FILA_SENHAS/$s"; then [ "$s" = "$n" ] && break; continue; fi # reserva: fora da disputa
      # a cabeça leve com contêiner que espera a exclusiva não segura a sem contêiner de trás (dono, 2026-11-22)
      if fila_cede "$FILA_SENHAS/$s" "$n"; then continue; fi
      [ -n "$cabeca" ] || cabeca=$s
      [ "$s" = "$n" ] && break
      pos=$((pos + 1))
    done
    vagas="-"
    if [ "$cabeca" = "$n" ]; then
      fila_limites
      leitura=$(fila_avaliar); vagas=${leitura%% *}
      read -r _ _ _ _ lr _ _ lmot <<< "$leitura"
      if [ "$fila_pesada" = 1 ]; then fila_janela_pesada "$n" "$lmot"; fi   # janela da leve (dono, 2026-11-22)
      if [ "$lmot" = - ]; then rm -f "$FILA_SENHAS/$n/motivo"; else echo "$lmot" > "$FILA_SENHAS/$n/motivo"; fi
      # a exclusiva é conferida de novo sob o lock de decisão: a leitura da memória leva segundos, e a outra pista
      # pode ter entrado nesse meio-tempo (dono, 2026-11-22)
      if [ "$vagas" -gt 0 ] && fila_decide_pega; then
        if [ -z "$(fila_exclusiva)" ]; then fila_pega_vaga "$lr" "$vagas" "$antigos"; fi
        fila_decide_solta
        [ -z "$fila_vaga" ] || break
      fi
    else
      rm -f "$FILA_SENHAS/$n/motivo"   # passou a ter alguém à frente: o motivo era de quando era cabeça
    fi
    if [ "$avisado" != true ]; then
      echo "fila do gate pesado: senha $n (à frente: $pos); rodando: $(cat "$FILA_LOCK/dono" "$FILA_LOCK".[0-9]*/dono 2>/dev/null | tr '\n' ';' || echo '-') — esperando…" >&2
      avisado=true
    fi
    sleep "$FILA_ESPERA_S"
  done
  echo "$$" > "$fila_vaga/pid"; echo "$dono $(date -u +%FT%TZ)" > "$fila_vaga/dono"
  echo "$fila_vaga" > "$FILA_SENHAS/$n/vaga"; rm -f "$FILA_SENHAS/$n/motivo"
  if [ "$fila_leve" = 1 ] && [ "$fila_pesada" != 1 ] && [ "$(fila_pesadas_vivas)" -gt 0 ]; then
    echo 'leve sem contêiner durante a pesada' > "$FILA_SENHAS/$n/durante"   # o --status mostra
  elif [ "$fila_leve" = 1 ] && [ "$fila_pesada" != 1 ] && fila_janela_aberta; then
    echo 'leve sem contêiner na janela da pesada' > "$FILA_SENHAS/$n/durante"
  fi
  [ "$fila_pesada" != 1 ] || fila_janela_fecha "$n" "a pesada entrou"
  # Leitura que abriu a vaga (Core, 2026-11-02: provar entradas sob pressão). Formato: chave=valor numa linha.
  read -r lv lc lsw lp lr lk lm _ <<< "$leitura"; vi=1; [ "$fila_vaga" = "$FILA_LOCK" ] || vi=${fila_vaga##*.}
  case "$lm" in -) lm=nao-lida ;; *[!0-9]*) ;; *) lm=${lm}MiB ;; esac   # «memória da VM: não lida» vai ao log (2026-11-22)
  linha="$(date -u +%FT%TZ) senha=$n dono=$dono vaga=$vi carga=$lc swap=$lsw% pressao=$lp rodando=$lr crdb=$lk vagas=$lv"
  linha="$linha memvm=$lm pesada=$fila_pesada docker=${REVENU_GATE_DOCKER:-0}"
  echo "$linha" > "$FILA_SENHAS/$n/leitura"; echo "$linha" >> "$FILA_LOG" 2>/dev/null || true
  # o comando filho herda a senha: se chamar a fila de novo, entra como reentrante (dono, 2026-11-22)
  if [ "$fila_leve" = 1 ]; then export REVENU_GATE_SENHA="leve:$n:$$"; else export REVENU_GATE_SENHA="banco:$n:$$"; fi
}

fila_tem_vaga() { # <leve|banco>: 0 se uma senha nova desta pista entraria agora; só lê, sem senha nem vaga.
  # Para o hook pre-push falhar rápido com a fila cheia (dono, 2026-11-22: o GitHub derruba o SSH do `git push` que
  # espera vaga; SocketOS, OnboardOS e FinanceOS). Conservadora: senha viva esperando nesta pista já é «sem vaga».
  # Vale o ambiente de quem pergunta (REVENU_GATE_DOCKER, REVENU_GATE_PESADA), como no fila_entrar.
  case "$1" in
    leve) fila_leve=1; FILA_LOCK=$FILA_LOCK_LEVE; FILA_SENHAS=${FILA_LOCK%.lock}.senhas; fila_pesada=0
          [ "${REVENU_GATE_PESADA:-0}" != 1 ] || fila_pesada=1 ;;
    banco) fila_leve=0; FILA_LOCK=$FILA_LOCK_BANCO; FILA_SENHAS=${FILA_LOCK%.lock}.senhas; fila_pesada=1 ;;
    *) echo "uso: --tem-vaga leve|banco" >&2; return 2 ;;
  esac
  fila_limites; [ "$fila_leve" = 0 ] || FILA_VAGAS_MAX=$FILA_LEVE_MAX
  local s l v r mot
  if [ "$fila_pesada" = 1 ] && ! fila_liberada; then echo "sem vaga $1: pesada sem liberação (DV 2397)"; return 1; fi
  for s in "$FILA_SENHAS"/*; do
    if [ ! -d "$s" ] || [ -f "$s/vaga" ] || ! fila_viva "$s"; then continue; fi
    echo "sem vaga $1: a senha ${s##*/} ($(cat "$s/dono" 2>/dev/null)) já espera"; return 1
  done
  l=$(fila_avaliar); v=${l%% *}; read -r _ _ _ _ r _ _ mot <<< "$l"
  if [ "${r:-0}" -lt "${v:-0}" ]; then echo "vaga $1 livre agora (rodando $r de $v)"; return 0; fi
  [ "$mot" != - ] || mot=""
  echo "sem vaga $1: rodando $r de $v${mot:+ ($mot)}"; return 1
}

fila_pega_vaga() { # <rodando> <vagas> <antigos>: tenta uma vaga desta pista; fila_vaga fica com ela, ou vazio
  # «vagas» é teto de gates RODANDO, não índice: com a vaga 2 ocupada e teto 1, a vaga 1 livre não abre
  # (MS-022 1.21: a 2ª prova do accountos entrou com VAGAS_MAX=1 em 2026-11-08). Abaixo do teto, qualquer vaga livre.
  local r=$1 vagas=$2 antigos=$3 v d
  for v in $(seq 1 "$FILA_VAGAS_MAX"); do
    [ "${r:-0}" -lt "$vagas" ] || break
    [ "$v" = 1 ] && [ "$antigos" -gt 0 ] && continue   # vaga 1 reservada às senhas antigas que esperam à frente
    d=$FILA_LOCK; [ "$v" = 1 ] || d=$FILA_LOCK.$v
    # vaga de dono morto: apaga e tenta a mesma (antes, `continue 2` voltava ao laço da cabeça; aqui há o lock de decisão)
    if ! mkdir "$d" 2>/dev/null; then fila_viva "$d" && continue; rm -rf "$d"; mkdir "$d" 2>/dev/null || continue; fi
    fila_vaga=$d; echo "$$" > "$d/pid"   # pid já aqui: a outra pista vê a vaga viva assim que o lock de decisão sai
    if [ "$fila_pesada" = 1 ]; then : > "$d/pesada"; fi
    return 0
  done
  return 0
}

fila_sair() {
  if [ "$fila_reentrante" = 1 ]; then return 0; fi   # aninhada: a senha e a vaga são do ancestral, que as solta
  fila_decide_solta   # sinal no meio da decisão não deixa o lock de decisão para trás
  if [ -n "$fila_vaga" ] && [ "$(cat "$fila_vaga/pid" 2>/dev/null)" = "$$" ]; then rm -rf "$fila_vaga"; fi
  fila_vaga=""
  if [ -n "$fila_senha" ] && [ "$fila_pesada" = 1 ]; then fila_janela_fecha "$fila_senha" 'a pesada saiu sem entrar'; fi
  if [ -n "$fila_senha" ]; then rm -rf "${FILA_SENHAS:?}/$fila_senha"; fila_senha=""; fi
}

FILA_CATEGORIAS=' dinheiro tenant assinatura pii contrato catalogo ' # as categorias do .ci-local/escopo.conf (CONVENTIONS § 6.10)

fila_nivel() { # <base> [<head>] [--previsto <arquivo>]: o nível; com --previsto, a lista escrita antes tem de bater
  local base="" head="" previsto="" saida
  while [ $# -gt 0 ]; do
    case "$1" in
      --previsto) previsto=${2:-}; [ -n "$previsto" ] || { echo amplia; echo "motivo: --previsto sem arquivo"; return 0; }; shift 2 ;;
      *) if [ -z "$base" ]; then base=$1; elif [ -z "$head" ]; then head=$1; else echo amplia; echo "motivo: argumento a mais: $1"; return 0; fi; shift ;;
    esac
  done
  saida=$(fila_nivel_de "$base" "${head:-HEAD}")
  # pacote que lê do disco o que mudou entra no escopo: na N0 e na N1 vai à lista (N0 vira N1), e o amplia o lista à parte
  local arquivos ld
  arquivos=$(git -c core.quotePath=false diff --no-renames --name-only "$base...${head:-HEAD}" 2>/dev/null) || arquivos=""
  ld=$(fila_le_disco_casa "${head:-HEAD}" "$arquivos")
  if [ -n "$ld" ]; then
    case $(printf '%s\n' "$saida" | sed -n 1p) in
      amplia) saida=$saida$'\n'$(printf '%s\n' "$ld" | sed 's/^/le-disco: /') ;;
      N0|N1) saida=$(echo N1; { printf '%s\n' "$saida" | sed 1d; printf '%s\n' "$ld"; } | sed '/^$/d' | LC_ALL=C sort -u) ;;
    esac
  fi
  if [ -n "$previsto" ]; then fila_previsto "$saida" "$previsto"; else printf '%s\n' "$saida"; fi
  return 0
}

fila_nivel_de() { # <base> <head>: N0 só com tudo na lista fechada da CONVENTIONS § 6.10; N1 com os pacotes; na dúvida, amplia
  local arquivos f go="" motivos
  arquivos=$(git -c core.quotePath=false diff --no-renames --name-only "$1...$2" 2>/dev/null) \
    || { echo amplia; echo "motivo: diff ilegível"; return 0; }
  [ -n "$arquivos" ] || { echo amplia; echo "motivo: diff vazio"; return 0; }
  # gatilho sensível do repo (dinheiro, tenant, assinatura, pii, contrato, catálogo): amplia com o motivo, antes de tudo
  motivos=$(fila_sensivel "$2" "$arquivos")
  if [ -n "$motivos" ]; then echo amplia; printf '%s\n' "$motivos" | awk '!v[$0]++ { print "motivo: " $0 }'; return 0; fi
  while IFS= read -r f; do
    case "$f" in
      .base/plans/milestones/*|.base/runbooks/evidence/*|.base/evidence/*|.base/flows/evidence/*) ;;
      .base/plans/*/milestones/*) # plano de um nível abaixo (`navigator-360/`), não mais fundo (dono, 2026-11-08, MS-022 1.23)
        local r=${f#.base/plans/}; r=${r#*/}
        case "$r" in milestones/*) ;; *) echo amplia; echo "motivo: plano fora da lista da N0: $f"; return 0 ;; esac ;;
      migrations/*|*/migrations/*|modules/shared/*|*/modules/shared/*|pkg/*|*/pkg/*|proto/*|*/proto/*) echo amplia; echo "motivo: caminho de base: $f"; return 0 ;;
      *.go) go="$go$f"$'\n' ;;
      *) echo amplia; echo "motivo: arquivo fora de .go: $f"; return 0 ;;
    esac
  done <<< "$arquivos"
  if [ -n "$go" ]; then fila_n1 "$go" "$2"; else echo N0; fi
  return 0
}

fila_sensivel() { # <head> <arquivos>: «<categoria>: <caminho>» para cada arquivo que casa um padrão do escopo do repo.
  # O escopo é .ci-local/escopo.conf no <head> (ou o arquivo de REVENU_GATE_ESCOPO): «<categoria> <padrão>» por linha, # comenta.
  # O padrão é glob do bash contra o caminho a partir da raiz (o * passa por /) e casa também tudo abaixo dele. Sem arquivo, nada.
  local conf cat pad resto f
  conf=$(fila_escopo_conf "$1") || { echo "escopo.conf ilegível: $REVENU_GATE_ESCOPO"; return 0; }
  while read -r cat pad resto; do
    case "$cat" in ''|'#'*) continue ;; esac
    if [ "$cat" = le-disco ]; then # «le-disco <pacote> <glob>»: não é categoria; só a forma é conferida aqui
      if [ -z "$pad" ] || [ "$(printf '%s\n' "$resto" | awk '{ print NF }')" != 1 ]; then echo "escopo.conf: linha inválida: le-disco $pad $resto"; fi
      continue
    fi
    case "$resto" in ''|'#'*) ;; *) echo "escopo.conf: linha inválida: $cat $pad $resto"; continue ;; esac
    case "$FILA_CATEGORIAS" in *" $cat "*) ;; *) echo "escopo.conf: categoria desconhecida: $cat"; continue ;; esac
    [ -n "$pad" ] || { echo "escopo.conf: $cat sem padrão"; continue; }
    pad=${pad%/}
    while IFS= read -r f; do
      # shellcheck disable=SC2053 # o padrão é glob de propósito
      if [[ $f == $pad || $f == $pad/* ]]; then echo "$cat: $f"; fi
    done <<< "$2"
  done <<< "$conf"
  return 0
}

fila_escopo_conf() { # <head>: o .ci-local/escopo.conf do <head>, ou o arquivo de REVENU_GATE_ESCOPO; sem arquivo, vazio
  if [ -n "${REVENU_GATE_ESCOPO:-}" ]; then cat "$REVENU_GATE_ESCOPO" 2>/dev/null; return; fi
  git show "$1:.ci-local/escopo.conf" 2>/dev/null || true
}

fila_le_disco() { # <head>: «<pacote> <glob>» de cada linha le-disco do escopo (CONVENTIONS § 6.10, MS-022 1.24)
  fila_escopo_conf "$1" | awk '$1 == "le-disco" && NF == 3 { print $2, $3 }'
}

fila_le_disco_casa() { # <head> <arquivos>: os pacotes le-disco cujo glob casa algum arquivo (o glob casa tudo abaixo dele)
  local pac pad f
  while read -r pac pad; do
    [ -n "$pac" ] || continue; pad=${pad%/}
    while IFS= read -r f; do
      # shellcheck disable=SC2053 # o padrão é glob de propósito
      if [[ $f == $pad || $f == $pad/* ]]; then echo "$pac"; break; fi
    done <<< "$2"
  done <<< "$(fila_le_disco "$1")" | LC_ALL=C sort -u
}

fila_leituras() { # portão (CONVENTIONS § 6.10, MS-022 1.24): arquivo de código que lê do disco com `..` no texto lê fora do
  # pacote; o pacote tem de estar em le-disco. Varre todos os .go do pacote (GoFiles, TestGoFiles, XTestGoFiles) e o TS/JS.
  # Saída: uma linha por pacote achado, «le-disco ok: <pacote>» ou «leitura não declarada: <pacote> (<arquivo>)»; rc 1 se houver
  # não declarada. Heurística: leitura que monta o caminho sem `..` (raiz achada pelo go.mod, por exemplo) não aparece.
  local raiz decl f d m rel pac vistos="" ruim=0 ok=0
  raiz=$(git rev-parse --show-toplevel 2>/dev/null) || { echo "fila: --leituras fora de repositório git" >&2; return 2; }
  decl=$(fila_le_disco HEAD | awk '{ print $1 }' | LC_ALL=C sort -u)
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    case "$f" in vendor/*|*/vendor/*|node_modules/*|*/node_modules/*) continue ;; esac
    case "$f" in
      *.go) grep -Eq 'os\.(ReadFile|ReadDir|Open|OpenFile|DirFS)\(|ioutil\.Read(File|Dir)\(|//go:embed' "$raiz/$f" || continue ;;
      *) grep -Eq 'readFileSync|readdirSync|readFile\(|readdir\(' "$raiz/$f" || continue ;;
    esac
    grep -Eq "[\"'\`]\.\.([/\"'\`])" "$raiz/$f" || continue
    d=$(dirname "$f"); pac=$d
    if [ "${f%.go}" != "$f" ]; then
      m=$d; while [ "$m" != . ] && [ ! -f "$raiz/$m/go.mod" ]; do m=$(dirname "$m"); done
      if [ -f "$raiz/$m/go.mod" ]; then
        rel=${d#"$m"}; rel=${rel#/}
        pac=$(cd "$raiz/$m" && go list -f '{{.ImportPath}}' "./$rel" 2>/dev/null) || pac=$d
        [ "$m" = . ] || pac="$m:$pac"
      fi
    fi
    case " $vistos " in *" $pac "*) continue ;; esac; vistos="$vistos $pac"
    if printf '%s\n' "$decl" | grep -Fxq -- "$pac"; then echo "le-disco ok: $pac"; ok=$((ok + 1))
    else echo "leitura não declarada: $pac ($f)"; ruim=$((ruim + 1)); fi
  done < <(git -C "$raiz" -c core.quotePath=false ls-files -- '*.go' '*.ts' '*.mts' '*.cts' '*.js' '*.mjs')
  echo "leituras: $ok declarada(s), $ruim não declarada(s)"
  [ "$ruim" = 0 ]
}

fila_previsto() { # <saída do nível> <arquivo>: a lista escrita antes (um pacote por linha, # comenta) contra a calculada
  local nivel calc prev mais menos
  nivel=$(printf '%s\n' "$1" | sed -n 1p)
  [ "$nivel" = amplia ] && { printf '%s\n' "$1"; return 0; }   # já amplia: o previsto não estreita nada
  prev=$(sed 's/#.*//; s/^[[:space:]]*//; s/[[:space:]]*$//; /^$/d' "$2" 2>/dev/null | LC_ALL=C sort -u) \
    || { echo amplia; echo "motivo: previsto ilegível: $2"; return 0; }
  [ -r "$2" ] || { echo amplia; echo "motivo: previsto ilegível: $2"; return 0; }
  calc=$(printf '%s\n' "$1" | sed '1d; /^$/d' | LC_ALL=C sort -u)   # N0: conjunto vazio
  mais=$(LC_ALL=C comm -23 <(printf '%s\n' "$calc" | sed '/^$/d') <(printf '%s\n' "$prev" | sed '/^$/d') | sed 's/^/+/' | tr '\n' ' ')
  menos=$(LC_ALL=C comm -13 <(printf '%s\n' "$calc" | sed '/^$/d') <(printf '%s\n' "$prev" | sed '/^$/d') | sed 's/^/-/' | tr '\n' ' ')
  if [ -z "$mais$menos" ]; then printf '%s\n' "$1"; return 0; fi
  echo amplia; echo "motivo: previsto diverge: $(printf '%s%s' "$mais" "$menos" | sed 's/ $//')"
  return 0
}

fila_modulo() { # <raiz> <diretório relativo>: o diretório (relativo) do go.mod mais próximo, "." na raiz; 1 fora de módulo
  local d=$2
  while :; do
    [ -f "$1/$d/go.mod" ] && { echo "$d"; return 0; }
    [ "$d" = . ] && return 1
    d=$(dirname "$d")
  done
}

fila_replace_de() { # <go.mod>: os alvos locais (./, ../, /) dos replace, um por linha
  awk '/^replace[ \t]*\(/ { b = 1; next } b && /^\)/ { b = 0; next }
       (b || /^replace[ \t]/) && /=>/ { s = $0; sub(/.*=>[ \t]*/, "", s); split(s, a, /[ \t]+/); if (a[1] ~ /^(\.|\/)/) print a[1] }' "$1" 2>/dev/null
}

fila_n1() { # <arquivos .go> <head>: N1 e os pacotes tocados com os dependentes, pelo `go list` de cada módulo da árvore.
  # Cada .go vai ao go.mod mais próximo; o fecho é por módulo. Saída: «N1» e um pacote por linha, com o prefixo
  # «<dir do módulo>:» quando o módulo não está na raiz (src:ledgeros/internal/x). Na dúvida, amplia.
  local raiz f d m rel pares="" mods o alvo absm usado tocados dirs grafo saida="" n p
  [ "$(git rev-parse "$2" 2>/dev/null)" = "$(git rev-parse HEAD 2>/dev/null)" ] \
    || { echo amplia; echo "motivo: head diferente de HEAD"; return 0; } # o go list lê a árvore
  raiz=$(git rev-parse --show-toplevel 2>/dev/null) || { echo amplia; echo "motivo: fora de repositório git"; return 0; }
  command -v go >/dev/null || { echo amplia; echo "motivo: go ausente"; return 0; }
  while IFS= read -r f; do
    [ -n "$f" ] || continue
    d=$(dirname "$f")
    m=$(fila_modulo "$raiz" "$d") || { echo amplia; echo "motivo: .go fora de módulo: $f"; return 0; }  # stack sem estratégia de N1
    if [ "$m" = . ]; then rel=$d; else rel=${d#"$m"}; rel=${rel#/}; [ -n "$rel" ] || rel=.; fi
    pares="$pares$m"$'\t'"$rel"$'\n'
  done <<< "$1"
  mods=$(printf '%s' "$pares" | cut -f1 | sort -u)
  if [ -f "$raiz/go.work" ] && [ "$(cd "$raiz" && git ls-files -- 'go.mod' '*/go.mod' | wc -l)" -gt 1 ]; then
    echo amplia; echo "motivo: go.work: os módulos se enxergam sem replace"; return 0
  fi
  # módulo tocado que outro módulo do repo puxa por replace local: o fecho cruzaria módulos; amplia
  while IFS= read -r m; do
    absm=$(cd "$raiz/$m" && pwd -P)
    while IFS= read -r o; do
      o=$(dirname "$o"); [ "$o" = "$m" ] && continue
      while IFS= read -r alvo; do
        [ -n "$alvo" ] || continue
        case "$alvo" in /*) usado=$alvo ;; *) usado=$raiz/$o/$alvo ;; esac
        if [ "$(cd "$usado" 2>/dev/null && pwd -P)" = "$absm" ]; then
          echo amplia; echo "motivo: módulo $m usado por $o via replace"; return 0
        fi
      done <<< "$(fila_replace_de "$raiz/$o/go.mod")"
    done <<< "$(cd "$raiz" && git ls-files -- 'go.mod' '*/go.mod')"
  done <<< "$mods"
  while IFS= read -r m; do
    dirs=$(printf '%s' "$pares" | awk -F'\t' -v m="$m" '$1 == m { print $2 }' | sort -u)
    # pacote apagado ou quebrado não resolve: a prova amplia, com o diretório (relativo à raiz do repo)
    tocados=""
    while IFS= read -r d; do
      p=$(cd "$raiz/$m" && go list -e -f '{{if not .Error}}{{.ImportPath}}{{end}}' "./$d" 2>/dev/null | sed '/^$/d')
      if [ -z "$p" ]; then
        if [ "$m" = . ]; then rel=$d; elif [ "$d" = . ]; then rel=$m; else rel=$m/$d; fi
        echo amplia; echo "motivo: pacote não resolve: $rel"; return 0
      fi
      tocados="$tocados$p"$'\n'
    done <<< "$dirs"
    # o fecho conta TestImports e XTestImports: o controle formal da AccountOS depende disso
    grafo=$(cd "$raiz/$m" && go list -e -f '{{.ImportPath}}{{range .Imports}} {{.}}{{end}}{{range .TestImports}} {{.}}{{end}}{{range .XTestImports}} {{.}}{{end}}' ./... 2>/dev/null) \
      || { echo amplia; echo "motivo: go list falhou"; return 0; }
    n=$(printf '%s\n' "$grafo" | awk -v t="$(printf '%s' "$tocados" | tr '\n' ' ')" '
      { for (i = 2; i <= NF; i++) rev[$i] = rev[$i] " " $1 }
      END { k = split(t, fila, " "); for (i = 1; i <= k; i++) s[fila[i]] = 1
            for (i = 1; i <= k; i++) { m = split(rev[fila[i]], d, " ")
              for (j = 1; j <= m; j++) if (!(d[j] in s)) { s[d[j]] = 1; fila[++k] = d[j] } }
            for (p in s) print p }')
    if [ "$m" != . ]; then n=$(printf '%s\n' "$n" | sed "s#^#$m:#"); fi
    saida="$saida$n"$'\n'
  done <<< "$mods"
  echo N1
  printf '%s' "$saida" | sed '/^$/d' | sort
  return 0
}

fila_passo_ok() { # <passo>: nome curto, sem barra
  case "$1" in ''|*[!A-Za-z0-9._-]*) echo "fila: passo inválido: '$1'" >&2; return 2 ;; esac
}

fila_repo() { # o repositório, pelo remoto (clones do mesmo repositório dividem as provas); sem remoto, a pasta
  local r; r=$(git config --get remote.origin.url 2>/dev/null | sed 's#\.git$##; s#.*[:/]\([^/]*\)/\([^/]*\)$#\1_\2#')
  echo "${r:-$(basename "$(git rev-parse --show-toplevel 2>/dev/null || pwd)")}"
}

fila_tools_lock() { # o tools.lock que vale: REVENU_GATE_TOOLS_LOCK, o instalado no repo, ou o do lado da fila (pelo link)
  local r f s t
  if [ -n "${REVENU_GATE_TOOLS_LOCK:-}" ]; then echo "$REVENU_GATE_TOOLS_LOCK"; return 0; fi
  r=$(git rev-parse --show-toplevel 2>/dev/null || true)
  for f in "$r/scripts/ci-local/tools.lock" "$r/scripts/ci-local-servico/tools.lock"; do
    if [ -n "$r" ] && [ -f "$f" ]; then echo "$f"; return 0; fi
  done
  s=${BASH_SOURCE[0]}
  if [ -L "$s" ]; then t=$(readlink "$s"); case "$t" in /*) s=$t ;; *) s=$(dirname "$s")/$t ;; esac; fi
  echo "$(dirname "$s")/tools.lock"
}

fila_go_versao() { # REVENU_GATE_GO_VERSAO, ou o go local e o GOTOOLCHAIN (lidos fora de módulo: nada baixa); com o go.mod na
  # chave, o toolchain efetivo fica determinado
  if [ -n "${REVENU_GATE_GO_VERSAO:-}" ]; then echo "$REVENU_GATE_GO_VERSAO"; return 0; fi
  command -v go >/dev/null 2>&1 || { echo sem-go; return 0; }
  (cd / && GOFLAGS='' go env GOVERSION GOTOOLCHAIN 2>/dev/null | tr '\n' ' ' | sed 's/ $//') || echo sem-go
}

fila_mods_da_entrada() { # <caminho…>: o diretório (a partir da raiz) do go.mod do HEAD mais próximo de cada caminho
  local p d rel
  for p in "$@"; do
    if [ -d "$p" ]; then d=$p; else d=$(dirname "$p"); fi
    rel=$(cd "$d" 2>/dev/null && git rev-parse --show-prefix) || continue
    rel=${rel%/}; [ -n "$rel" ] || rel=.
    while :; do
      if [ "$rel" = . ]; then git cat-file -e HEAD:go.mod 2>/dev/null && echo .; break; fi
      git cat-file -e "HEAD:$rel/go.mod" 2>/dev/null && { echo "$rel"; break; }
      rel=$(dirname "$rel")
    done
  done | sort -u
}

fila_hash() { # <passo> <caminho…>: chave de idempotência (§ 6.7): árvore do HEAD nos caminhos, go.mod e go.sum do módulo de
  # cada caminho, go version, tools.lock e REVENU_GATE_ENTRADA (entrada a mais)
  local passo=$1 raiz mods m lock sujo a pac pad f ld="" lidos=""; shift; fila_passo_ok "$passo" || return 2
  [ $# -gt 0 ] || { echo "fila: --hash pede ao menos um caminho de entrada" >&2; return 2; }
  git rev-parse HEAD >/dev/null 2>&1 || { echo "fila: --hash fora de repositório git" >&2; return 1; }
  raiz=$(git rev-parse --show-toplevel)
  # «le-disco:<pacote>» como caminho: entram os arquivos do HEAD que casam os globs declarados para ele (§ 6.7)
  local reais=()
  for a in "$@"; do
    case "$a" in
      le-disco:*) pac=${a#le-disco:}
        pad=$(fila_le_disco HEAD | awk -v p="$pac" '$1 == p { print $2 }')
        [ -n "$pad" ] || { echo "fila: $pac sem linha le-disco no escopo.conf" >&2; return 2; }
        ld=$ld$pad$'\n' ;;
      *) reais+=("$a") ;;
    esac
  done
  if [ -n "$ld" ]; then
    while IFS= read -r f; do
      while IFS= read -r pad; do
        [ -n "$pad" ] || continue; pad=${pad%/}
        # shellcheck disable=SC2053 # o padrão é glob de propósito
        if [[ $f == $pad || $f == $pad/* ]]; then lidos=$lidos$f$'\n'; break; fi
      done <<< "$ld"
    done < <(git -C "$raiz" -c core.quotePath=false ls-tree -r --name-only --full-name HEAD)
  fi
  set -- ${reais[@]+"${reais[@]}"}
  mods=""; sujo=""; lock=$(fila_tools_lock)
  if [ $# -gt 0 ]; then mods=$(fila_mods_da_entrada "$@"); sujo=$(git status --porcelain -- "$@" 2>/dev/null); fi
  while IFS= read -r f; do if [ -n "$f" ]; then sujo=$sujo$(git -C "$raiz" status --porcelain -- "$f" 2>/dev/null); fi; done <<< "$lidos"
  while IFS= read -r m; do
    [ -n "$m" ] || continue
    sujo=$sujo$(git -C "$raiz" status --porcelain -- "$m/go.mod" "$m/go.sum" 2>/dev/null)
  done <<< "$mods"
  case "$lock" in "$raiz"/*) sujo=$sujo$(git -C "$raiz" status --porcelain -- "${lock#"$raiz"/}" 2>/dev/null) ;; esac
  [ -z "$sujo" ] || { echo "fila: entrada com mudança fora do commit; sem hash" >&2; return 1; }
  {
    echo "passo=$passo"; echo "entrada=${REVENU_GATE_ENTRADA:-}"; echo "go=$(fila_go_versao)"
    if [ -f "$lock" ]; then sed 's/#.*//; /^[[:space:]]*$/d; s/^/tools.lock /' "$lock"; else echo "tools.lock ausente"; fi
    while IFS= read -r m; do
      [ -n "$m" ] || continue
      echo "mod $m $(git -C "$raiz" rev-parse -q --verify "HEAD:$m/go.mod" 2>/dev/null || echo -) $(git -C "$raiz" rev-parse -q --verify "HEAD:$m/go.sum" 2>/dev/null || echo -)"
    done <<< "$mods"
    if [ $# -gt 0 ]; then git ls-tree -r --full-name HEAD -- "$@"; fi   # caminho da raiz: a chave não depende da pasta de onde se chama
    while IFS= read -r f; do if [ -n "$f" ]; then git -C "$raiz" ls-tree --full-name HEAD -- "$f"; fi; done <<< "$lidos"
  } | shasum -a 256 | cut -c1-16
}

fila_prova() { # <passo> <hash> [<caminho…>]: grava o passo verde com o head de origem; árvore suja não grava
  fila_passo_ok "$1" || return 2
  if printf '%s\n' "$1" | grep -E -q "$FILA_NUNCA"; then echo "fila: $1 nunca reaproveita; nada gravado" >&2; return 0; fi
  local passo=$1 hash=$2 sujo; shift 2
  if [ $# -gt 0 ]; then sujo=$(git status --porcelain -- "$@" 2>/dev/null); else sujo=$(git status --porcelain --untracked-files=no 2>/dev/null); fi
  [ -z "$sujo" ] || { echo "fila: árvore suja; prova de $passo não gravada" >&2; return 1; }
  local d; d=$FILA_ESTADO/provas/$(fila_repo)/$passo; mkdir -p "$d"
  echo "head $(git rev-parse --short HEAD 2>/dev/null) em $(date -u +%FT%TZ)${REVENU_GATE_NIVEL_TESTE:+ nivel=$REVENU_GATE_NIVEL_TESTE}" > "$d/$hash"
}

fila_reaproveita() { # <passo> <hash>: 0 e a origem, se a prova desse hash existe e o passo pode reaproveitar
  fila_passo_ok "$1" || return 2
  if printf '%s\n' "$1" | grep -E -q "$FILA_NUNCA"; then echo "roda: $1 nunca reaproveita"; return 1; fi
  local f; f=$FILA_ESTADO/provas/$(fila_repo)/$1/$2
  [ -f "$f" ] || { echo "roda: sem prova de $1 com o hash $2"; return 1; }
  echo "reaproveita: $1 $2, $(cat "$f")"
}

fila_chave() { printf '%s\n' "$*" | shasum -a 256 | cut -c1-12; } # <dono> <comando…>

fila_bloqueio() { # <chave>: 0 se pode entrar; 3 na DLQ; 75 com o breaker aberto (sem veredito)
  if [ "$FILA_DLQ" = 1 ] && [ -f "$FILA_ESTADO/dlq/$1" ]; then
    echo "fila: comando na DLQ ($(cat "$FILA_ESTADO/dlq/$1")); corrija e tire com --dlq-tira $1" >&2; return 3
  fi
  if [ -f "$FILA_ESTADO/ambiente" ]; then
    local t; t=$(sed -n 1p "$FILA_ESTADO/ambiente")
    if [ $(( $(date +%s) - ${t:-0} )) -lt "$FILA_BREAKER_S" ]; then
      echo "fila: ambiente marcado ($(sed -n 2p "$FILA_ESTADO/ambiente")); sem veredito: tente de novo depois ou --breaker-fecha" >&2
      return 75
    fi
  fi
  return 0
}

fila_registrar() { # <chave> <dono> <rc> <comando…>: conta falha para a DLQ e falha de ambiente para o breaker
  local chave=$1 dono=$2 rc=$3 n; shift 3
  mkdir -p "$FILA_ESTADO/falhas" "$FILA_ESTADO/dlq"
  if [ "$rc" = 0 ]; then rm -f "$FILA_ESTADO/falhas/$chave" "$FILA_ESTADO/ambiente" "$FILA_ESTADO/ambiente.n"; return 0; fi
  if [ "$rc" = 127 ]; then # binário ausente: nem veredito do código nem falha de ambiente
    echo "fila: comando não encontrado (rc 127); instale o binário ou troque o comando. Não conta para a DLQ nem para o breaker" >&2
    return 0
  fi
  if [ "$rc" = 124 ] || [ "$rc" = 75 ] || [ "$rc" -ge 128 ]; then # timeout, ambiente declarado ou sinal: não é veredito
    n=$(( $(cat "$FILA_ESTADO/ambiente.n" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$FILA_ESTADO/ambiente.n"
    echo "fila: falha de ambiente (rc $rc), sem veredito; não conta para a DLQ" >&2
    if [ "$n" -ge "$FILA_BREAKER_MAX" ]; then
      printf '%s\n%s\n' "$(date +%s)" "rc $rc de $dono, $n falhas de ambiente seguidas" > "$FILA_ESTADO/ambiente"
      echo "fila: breaker aberto por $FILA_BREAKER_S s" >&2
    fi
    return 0
  fi
  [ "$FILA_DLQ" = 1 ] || return 0
  n=$(( $(cat "$FILA_ESTADO/falhas/$chave" 2>/dev/null || echo 0) + 1 )); echo "$n" > "$FILA_ESTADO/falhas/$chave"
  if [ "$n" -ge "$FILA_DLQ_MAX" ]; then
    echo "$dono · rc $rc · $n falhas · $(date -u +%FT%TZ) · $*" > "$FILA_ESTADO/dlq/$chave"
    echo "fila: $n falhas seguidas; comando na DLQ ($chave)" >&2
  fi
}

fila_guardar_conteiner() { # [<contêiner…>]: fim do log e .State de quem saiu com código ≠ 0 ou OOMKilled; sem argumento,
  # todos os contêineres de gate parados (label revenu.gate.dono). O trap do módulo chama antes do rm (Core, 2026-11-08: dois
  # CRDB caíram na migração e o trap apagou antes de alguém ver o log).
  command -v docker >/dev/null 2>&1 || return 0
  local c estado nome d=$FILA_ESTADO/conteineres
  # shellcheck disable=SC2046 # id de contêiner não tem espaço
  [ $# -gt 0 ] || set -- $(docker ps -aq --filter label=revenu.gate.dono --filter status=exited 2>/dev/null)
  for c in "$@"; do
    estado=$(docker inspect -f '{{.State.ExitCode}} {{.State.OOMKilled}}' "$c" 2>/dev/null) || continue
    if [ "$estado" = "0 false" ]; then continue; fi
    mkdir -p "$d"; nome=$(printf '%s' "$c" | tr -c 'A-Za-z0-9_.-' _ | cut -c1-40)
    { docker inspect -f '{{.Name}} {{.Config.Image}} {{json .State}}' "$c"
      docker logs --tail "${REVENU_GATE_LOG_LINHAS:-200}" "$c"; } > "$d/$(date +%Y%m%dT%H%M%S)-$nome.log" 2>&1 || true
    echo "fila: contêiner $c saiu com «${estado}» (código, OOMKilled); guardado em $d" >&2
  done
  return 0
}

fila_matar_arvore() { # <pid> <sinal>
  local f; for f in $(pgrep -P "$1" 2>/dev/null); do fila_matar_arvore "$f" "$2"; done
  kill "-$2" "$1" 2>/dev/null || true
}

if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  set -euo pipefail
  case "${1:-}" in
    --nivel) : "${2:?base}"; shift; fila_nivel "$@"; exit 0 ;;
    --hash) shift; fila_hash "$@"; exit ;;
    --prova) : "${2:?passo}" "${3:?hash}"; shift; fila_prova "$@"; exit ;;
    --reaproveita) fila_reaproveita "${2:?passo}" "${3:?hash}"; exit ;;
    --status) fila_status; exit 0 ;;
    --tem-vaga) fila_tem_vaga "${2:-}"; exit ;;
    --leituras) fila_leituras; exit ;;
    --dlq) cat "$FILA_ESTADO"/dlq/* 2>/dev/null || true; exit 0 ;;
    --dlq-tira) rm -f "$FILA_ESTADO/dlq/${2:?chave}" "$FILA_ESTADO/falhas/$2"; exit 0 ;;
    --breaker-fecha) rm -f "$FILA_ESTADO/ambiente" "$FILA_ESTADO/ambiente.n"; exit 0 ;;
    --guardar-conteiner) shift; fila_guardar_conteiner "$@"; exit 0 ;;
  esac
  [ $# -ge 2 ] || { echo "uso: $0 <dono> <comando…>" >&2; exit 2; }
  dono=$1; shift
  trap fila_sair EXIT; trap 'exit 130' INT TERM
  command -v "$1" >/dev/null 2>&1 || { echo "fila: '$1' não existe nesta máquina; recusado antes de pegar a vez (rc 127)" >&2; exit 127; }
  # aninhada na senha de um ancestral: roda direto, com exec, sem senha, DLQ nem timeout próprios (dono, 2026-11-22)
  if fila_reentrada "$dono"; then trap - EXIT INT TERM; exec "$@"; fi
  fila_reentrada_vista=1
  chave=$(fila_chave "$dono" "$@"); fila_bloqueio "$chave" || exit $?
  fila_entrar "$dono"
  "$@" &
  filho=$!; inicio=$(date +%s); rc=""; volta=0
  while kill -0 "$filho" 2>/dev/null; do
    volta=$((volta + 1))
    if [ "$fila_leve" = 1 ] && [ $((volta % FILA_LEVE_CONFERE)) = 0 ]; then
      motivo=$(fila_confere_leve "$dono" "$filho" "$((inicio - 1))")
      if [ -n "$motivo" ]; then
        echo "fila do gate pesado: $dono declarou sem banco, mas a rodada subiu banco ($motivo) — encerrando (rc 66)" >&2
        fila_matar_arvore "$filho" TERM; sleep 2; fila_matar_arvore "$filho" KILL
        wait "$filho" 2>/dev/null || true
        rc=66; break
      fi
    fi
    if [ $(( $(date +%s) - inicio )) -ge "$FILA_TIMEOUT_S" ]; then
      echo "fila do gate pesado: $dono passou de $FILA_TIMEOUT_S s — encerrando e soltando a vaga" >&2
      fila_matar_arvore "$filho" TERM; sleep 20; fila_matar_arvore "$filho" KILL
      wait "$filho" 2>/dev/null || true
      rc=124; break
    fi
    sleep "$FILA_VOLTA_S"
  done
  if [ -z "$rc" ]; then set +e; wait "$filho"; rc=$?; set -e; fi
  fila_registrar "$chave" "$dono" "$rc" "$@"
  exit "$rc"
fi
