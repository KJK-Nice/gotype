#!/usr/bin/env bash
cd /workspace
export GOTYPE_DATA_DIR=/tmp/gotype-demo
export TERM=xterm-256color
exec go run ./cmd/gotype
