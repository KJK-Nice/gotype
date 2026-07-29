# mtype

Monkeytype-style typing test for the terminal. Built with Go + Bubble Tea.

## Install

```bash
go install github.com/kjkusap/monkeytype-clone/cmd/mtype@latest
```

Or from this repo:

```bash
go run ./cmd/mtype
```

## Usage

```
mtype
```

1. Pick **time** or **words** mode (`t` / `w`, or ↑↓)
2. Pick duration / word count (tab to value, ↑↓ to change)
3. Enter to start — type the prompt
4. See WPM chart, accuracy, and breakdown on finish

### Keys

| Key | Action |
|-----|--------|
| ↑↓ / j k | Change mode or value |
| tab | Switch focus / restart test |
| t / w | Time / words mode |
| enter / space | Start (menu) or next test (results) |
| esc | Back to menu |
| q / ctrl+c | Quit |

## Modes

- **time** — 15 / 30 / 60 / 120 seconds
- **words** — 10 / 25 / 50 / 100 words

Metrics follow Monkeytype conventions: WPM from correct chars÷5÷minutes, accuracy from correct÷typed.

## Play over SSH

`mtype-ssh` serves the same TUI over SSH (Charm Wish). Auth is open — any user/password.

### Local

```bash
go run ./cmd/mtype-ssh
# default listen 0.0.0.0:2222

ssh play@localhost -p 2222
# any username works; password can be empty/anything
```

Optional: set `PORT` and/or `SSH_HOST_KEY` (PEM private key) so the host key stays stable across restarts.

```bash
export SSH_HOST_KEY="$(cat ./host_ed25519)"
PORT=2222 go run ./cmd/mtype-ssh
```

### Railway

Deploy from this repo (Dockerfile builds `mtype-ssh`). Enable a **TCP Proxy** to internal port `2222`, and set secret `SSH_HOST_KEY` to a durable ed25519 PEM (generate once; redeploys keep the same fingerprint).

```bash
# generate once, paste PEM into Railway variable SSH_HOST_KEY
ssh-keygen -t ed25519 -f host_ed25519 -N "" -C mtype-ssh
```

Connect (Railway assigns a high public port — not 22):

```bash
ssh play@<railway-tcp-domain> -p <assigned-port>
```
