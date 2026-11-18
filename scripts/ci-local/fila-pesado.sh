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
# Dono, select no DV em 2026-11-06: até 3 gates com banco juntos, também em warn (revoga «um por vez» de 05/11); critical segue 1.
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
  local k v
  while IFS='=' read -r k v; do
    case "$v" in ''|*[!0-9]*) continue ;; esac
    case "$k" in
      VAGAS_MAX) FILA_VAGAS_MAX=$v ;; CARGA_2) FILA_CARGA_2=$v ;; CARGA_0) FILA_CARGA_0=$v ;;
      SWAP_MAX) FILA_SWAP_MAX=$v ;; VAGAS_WARN) FILA_VAGAS_WARN=$v ;; CRDB_MAX) FILA_CRDB_MAX=$v ;;
      LEVE_MAX) FILA_LEVE_MAX=$v ;; LEVE_WARN) FILA_LEVE_WARN=$v ;;
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
  command -v docker >/dev/null 2>&1 && c=$(docker ps --filter label=revenu.gate.dono --format '{{.Image}}' 2>/dev/null | grep -c -i cockroach || true)
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
  # depois do início, com o label revenu.gate.dono do dono ou sem label, desde que nenhum gate com banco esteja vivo
  # para disputar a autoria. Quem sobe banco declarado leve é encerrado.
  [ "$fila_leve" = 1 ] || return 0
  local dono=$1 pid=$2 inicio=$3 p cmd
  for p in $(fila_descendentes "$pid"); do
    cmd=$(ps -o command= -p "$p" 2>/dev/null) || continue
    case "$cmd" in
      *"cockroach start"*|*kafka-server-start*) echo "processo $p: ${cmd:0:120}"; return 0 ;;
      # docker só conta quando sobe banco: o act, o hadolint e o dockle rodam `docker run` sem banco (MS-022 1.26)
      *"docker run"*|*"docker create"*|*"docker compose"*|*docker-compose*)
        case "$(printf '%s' "$cmd" | tr '[:upper:]' '[:lower:]')" in
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
    case "$(printf '%s' "$rot" | tr '[:upper:]' '[:lower:]')" in ''|"$d") ;; *) continue ;; esac
    ts=$(date -j -f '%Y-%m-%d %H:%M:%S %z' "${criado:0:25}" +%s 2>/dev/null || date -d "${criado:0:25}" +%s 2>/dev/null) || continue # «2026-11-08 19:15:38 +0100 WAT»
    [ "$ts" -ge "$inicio" ] && { echo "contêiner $id ($img) criado durante a rodada"; return 0; }
  done < <(docker ps --format '{{.ID}}|{{.Image}}|{{.Label "revenu.gate.dono"}}|{{.CreatedAt}}' 2>/dev/null || true)
  return 0
}

fila_descendentes() { # <pid>: os descendentes, um por linha
  local f; for f in $(pgrep -P "$1" 2>/dev/null); do echo "$f"; fila_descendentes "$f"; done
}

fila_avaliar() { # «vagas carga swap% pressão rodando crdb»: as vagas e as leituras que as decidiram, numa linha
  local c p sw r k v; c=$(fila_carga); p=$(fila_pressao); sw=$(fila_swap_pct); r=$(fila_rodando); k=$(fila_crdb_vivos)
  if [ "$fila_leve" = 1 ]; then v=$(fila_vagas_leve); else v=$(fila_vagas_de "$c" "$p" "$sw" "$r" "$k"); fi
  echo "$v $c $sw $p $r $k"
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
    /^#/ || NR == 1 && $1 == "ts_utc" { next }
    NF >= 4 { k = $2 SUBSEP $3
      if ($4 == "inicio") { ini[k] = $1; ordem[++n] = k } else { delete ini[k]; fim++; if ($4 == "falha") f++; if ($4 == "estourou") e++ } }
    END {
      for (i = n; i >= 1; i--) if (ordem[i] in ini) { split(ordem[i], a, SUBSEP); c = a[1] " " a[2] " desde " ini[ordem[i]]; break }
      if (c == "") c = "-"
      if (r == "") r = "terminados=" fim + 0 " falhas=" f + 0
      if (e > 0) r = r " estourou=" e " ESTOUROU" # passo cortado pelo teto: ⚠️, não vermelho (CONVENTIONS § 6.6)
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
        # teto de parede do nível declarado (N1 15 min, N2 45 min; CONVENTIONS § 6.10): passou, vira investigação com dono
        case $(cat "$dir/$s/nivel" 2>/dev/null) in
          N1) [ $(( agora - desde )) -gt "$FILA_TETO_N1_S" ] && estado="$estado TETO>N1" ;;
          N2) [ $(( agora - desde )) -gt "$FILA_TETO_N2_S" ] && estado="$estado TETO>N2" ;;
        esac
      else
        desde=$(stat -f %m "$dir/$s/pid" 2>/dev/null || stat -c %Y "$dir/$s/pid" 2>/dev/null || echo "$agora"); estado="espera há $(( (agora - desde) / 60 )) min"
        [ $(( agora - desde )) -gt "$FILA_ESPERA_ALERTA_S" ] && estado="$estado ESPERA>30min"
        fila_barrada "$dir/$s" && estado="$estado RESERVA"
      fi
      prog=-; [ -f "$dir/$s/progresso" ] && prog=$(fila_progresso_resumo "$(cat "$dir/$s/progresso")")
      printf '%s\t%s\t%s\t%s\t%s\t%s\n' "$pista" "$s" "$(cat "$dir/$s/dono" 2>/dev/null)" "$pid" "$estado" "$prog"
    done
  done
}

fila_entrar() { # <dono>
  if [ "${REVENU_GATE_SEM_BANCO:-0}" = 1 ]; then # pista própria: outra pasta de senhas e de vagas, sem esperar a cabeça com banco
    fila_leve=1; FILA_LOCK=$FILA_LOCK_LEVE; FILA_SENHAS=${FILA_LOCK%.lock}.senhas; FILA_VAGAS_MAX=$FILA_LEVE_MAX
  fi
  local dono=$1 n s pos cabeca vagas v d antigos avisado=false leitura="" linha lv lc lsw lp lr lk vi
  mkdir -p "$FILA_SENHAS"
  while :; do
    n=$(fila_nova_senha "$dono")
    mkdir "$FILA_SENHAS/$n" 2>/dev/null && break
  done
  fila_senha=$n; echo "$$" > "$FILA_SENHAS/$n/pid"; echo "$dono" > "$FILA_SENHAS/$n/dono"; : > "$FILA_SENHAS/$n/cabeca"
  # progresso.tsv da prova (CONVENTIONS § 6.6): o caminho fica na senha, e o --status lê dali o passo e o pacote em curso
  case ${REVENU_GATE_NIVEL_PROVA:-} in N1|N2) echo "$REVENU_GATE_NIVEL_PROVA" > "$FILA_SENHAS/$n/nivel" ;; esac
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
      [ -n "$cabeca" ] || cabeca=$s
      [ "$s" = "$n" ] && break
      pos=$((pos + 1))
    done
    vagas="-"
    if [ "$cabeca" = "$n" ]; then
      fila_limites
      leitura=$(fila_avaliar); vagas=${leitura%% *}
      # «vagas» é teto de gates RODANDO, não índice: com a vaga 2 ocupada e teto 1, a vaga 1 livre não abre
      # (MS-022 1.21: a 2ª prova do accountos entrou com VAGAS_MAX=1 em 2026-11-08). Abaixo do teto, qualquer vaga livre.
      read -r _ _ _ _ lr _ <<< "$leitura"
      for v in $(seq 1 "$FILA_VAGAS_MAX"); do
        [ "${lr:-0}" -lt "$vagas" ] || break
        [ "$v" = 1 ] && [ "$antigos" -gt 0 ] && continue   # vaga 1 reservada às senhas antigas que esperam à frente
        d=$FILA_LOCK; [ "$v" = 1 ] || d=$FILA_LOCK.$v
        if mkdir "$d" 2>/dev/null; then fila_vaga=$d; break 2; fi
        fila_viva "$d" || { rm -rf "$d"; continue 2; }
      done
    fi
    if [ "$avisado" != true ]; then
      echo "fila do gate pesado: senha $n (à frente: $pos); rodando: $(cat "$FILA_LOCK/dono" "$FILA_LOCK".[0-9]*/dono 2>/dev/null | tr '\n' ';' || echo '-') — esperando…" >&2
      avisado=true
    fi
    sleep "$FILA_ESPERA_S"
  done
  echo "$$" > "$fila_vaga/pid"; echo "$dono $(date -u +%FT%TZ)" > "$fila_vaga/dono"
  echo "$fila_vaga" > "$FILA_SENHAS/$n/vaga"
  # Leitura que abriu a vaga (Core, 2026-11-02: provar entradas sob pressão). Formato: chave=valor numa linha.
  read -r lv lc lsw lp lr lk <<< "$leitura"; vi=1; [ "$fila_vaga" = "$FILA_LOCK" ] || vi=${fila_vaga##*.}
  linha="$(date -u +%FT%TZ) senha=$n dono=$dono vaga=$vi carga=$lc swap=$lsw% pressao=$lp rodando=$lr crdb=$lk vagas=$lv"
  echo "$linha" > "$FILA_SENHAS/$n/leitura"; echo "$linha" >> "$FILA_LOG" 2>/dev/null || true
}

fila_sair() {
  if [ -n "$fila_vaga" ] && [ "$(cat "$fila_vaga/pid" 2>/dev/null)" = "$$" ]; then rm -rf "$fila_vaga"; fi
  fila_vaga=""
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
  echo "head $(git rev-parse --short HEAD 2>/dev/null) em $(date -u +%FT%TZ)" > "$d/$hash"
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
