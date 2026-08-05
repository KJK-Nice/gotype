#!/usr/bin/env bash
# Drives gotype through progression, claim, multi, and theme for asciinema.
set -euo pipefail
cd /workspace
export GOTYPE_DATA_DIR="${GOTYPE_DATA_DIR:-/tmp/gotype-demo-cast}"
export TERM="${TERM:-xterm-256color}"

coproc GOTYPE ( go run ./cmd/gotype )
sleep 2.5

send() {
	printf '%s' "$1" >&"${GOTYPE[1]}"
	sleep "${2:-1.4}"
}

# progression tabs
send 'i' 1.6
send 's' 1.6
send 'p' 1.8
send 'e' 1.6
printf '\e' >&"${GOTYPE[1]}" # esc — close progression
sleep 1

# claim overlay
send 'c' 1.6
printf '\e' >&"${GOTYPE[1]}"
sleep 1

# multiplayer menu
send 'm' 1.6
printf '\e' >&"${GOTYPE[1]}"
sleep 1

# theme cycle + quit
send 'u' 1
send 'q' 0.6

wait "${GOTYPE_PID}" 2>/dev/null || true
