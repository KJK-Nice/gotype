#!/usr/bin/env bash
# Automated walkthrough of the new UI surfaces for screen recording.
set -euo pipefail
export GOTYPE_DATA_DIR="${GOTYPE_DATA_DIR:-/tmp/gotype-demo}"
export TERM=xterm-256color

sleep 2
# progression tabs from solo menu
printf 'i'
sleep 1.2
printf 's'
sleep 1.2
printf 'p'
sleep 1.2
printf 'e'
sleep 1.2
printf '\033'  # esc — close progression
sleep 0.8
# claim overlay
printf 'c'
sleep 1.5
printf '\033'
sleep 0.8
# multiplayer menu
printf 'm'
sleep 1.5
printf '\033'
sleep 0.8
# theme cycle
printf 'u'
sleep 0.8
printf 'q'
