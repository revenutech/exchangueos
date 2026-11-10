#!/usr/bin/env bash
# Lint of the pattern catalogue contract (.base/plans/01-architecture/patterns/TEMPLATE.md, MS-024k 1.2).
#
# Usage: scripts/lint-patterns.sh <catalogue.md>...
#
# Unit: one pattern = one `## FX-<FAM>-NNN` or `### FX-<FAM>-NNN` section of a catalogue file, up to the next
# heading of the same or higher level. A catalogue file holds many patterns, so each section is checked alone.
# Each pattern must have, in this order, the paragraphs `**Context:**`, `**Problem:**`, `**Solution:**`,
# `**Example:**`, `**Anti-pattern:**` and `**Related:**`; at least one fenced code block; a non-empty
# Anti-pattern; and at least one cross-link in Related (`FX-<FAM>-NNN` other than itself, or `RN_FX_NNN`).
# A file with no pattern section fails. Exit 0 when every pattern passes, 1 otherwise, 2 on usage error.
set -u

if [ "$#" -eq 0 ]; then
  echo "usage: $0 <catalogue.md>..." >&2
  exit 2
fi

rc=0
for f in "$@"; do
  if [ ! -r "$f" ]; then
    echo "$f: unreadable" >&2
    rc=1
    continue
  fi
  awk -v file="$f" '
    function flush(   i, want, pos, n, missing) {
      if (id == "") return
      n = split("Context Problem Solution Example Anti-pattern Related", want, " ")
      pos = 0
      for (i = 1; i <= n; i++) {
        if (!(want[i] in seen)) { err(want[i] " section missing"); continue }
        if (seen[want[i]] < pos) err(want[i] " out of order")
        pos = seen[want[i]]
      }
      if (fences < 2) err("no fenced code example")
      if (("Anti-pattern" in seen) && antitext == "") err("empty Anti-pattern")
      if (("Related" in seen) && !xlink) err("Related has no cross-link (FX-<FAM>-NNN or RN_FX_NNN)")
    }
    function err(msg) { printf "%s:%d: %s: %s\n", file, start, id, msg; bad = 1 }
    function reset() { delete seen; fences = 0; antitext = ""; xlink = 0; insec = ""; line = 0 }
    /^##+ / {
      lvl = index($0, " ") - 1
      if (match($0, /FX-[A-Z]+-[0-9]+/)) {
        if (id != "") flush()
        reset(); id = substr($0, RSTART, RLENGTH); start = NR; plevel = lvl; found = 1
        next
      }
      if (id != "" && lvl <= plevel) { flush(); id = "" }
      next
    }
    id == "" { next }
    /^[ \t]*(```|~~~)/ { fences++ }
    {
      line++
      if (match($0, /^\*\*(Context|Problem|Solution|Example|Anti-pattern|Related):\*\*/)) {
        insec = substr($0, 3, RLENGTH - 5)
        if (!(insec in seen)) seen[insec] = line
        rest = substr($0, RLENGTH + 1)
      } else rest = $0
      if (insec == "Anti-pattern" && rest ~ /[^ \t]/) antitext = antitext rest
      if (insec == "Related") {
        tmp = rest
        while (match(tmp, /FX-[A-Z]+-[0-9]+|RN_FX_[0-9]+/)) {
          if (substr(tmp, RSTART, RLENGTH) != id) xlink = 1
          tmp = substr(tmp, RSTART + RLENGTH)
        }
      }
    }
    END {
      flush()
      if (!found) { printf "%s: no FX-<FAM>-NNN pattern section\n", file; bad = 1 }
      exit bad
    }
  ' "$f" || rc=1
done
exit "$rc"
