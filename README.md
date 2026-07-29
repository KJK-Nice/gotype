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

## Multiplayer (SSH)

Over `mtype-ssh`, press **m** on the menu:

1. **c** create room → share 4-letter code
2. Friend **j** join → type code
3. Host **s** / enter to start (need 2–4 players)
4. 3s countdown → same prompt → live WPM bars
5. Podium when time ends (or all finish)

SSH username = display name (`ssh alice@host -p PORT`).

Solo `go run ./cmd/mtype` has no multiplayer (no shared hub).

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
# generate once
ssh-keygen -t ed25519 -f host_ed25519 -N "" -C mtype-ssh

railway login
railway init   # or: railway link
railway up
railway variables set SSH_HOST_KEY="$(cat host_ed25519)"
railway variables set PORT=2222
# Dashboard → service → Settings → Networking → TCP Proxy → port 2222
```

Connect (Railway assigns a high public port — not 22):

```bash
ssh play@sakura.proxy.rlwy.net -p 58372
```

(Replace host/port if your Railway TCP proxy changed.)
