#!/usr/bin/env bash
set -euo pipefail
sleep 3
WID=$(xdotool search --name 'gotype' | head -1)
xdotool windowactivate --sync "$WID"
sleep 0.5
for key in i s p e Escape c Escape m Escape u q; do
  xdotool key --window "$WID" "$key"
  sleep 1.4
done
