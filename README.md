# gotype

Typing races for the terminal. Built with Go + Bubble Tea.

Play at [gotype.fun](https://gotype.fun) — or over SSH:

```bash
ssh play@game.gotype.fun -p 58372
```

DNS for SSH host (registrar — **not** a Railway HTTP domain):

| Type | Name | Content |
|------|------|---------|
| CNAME | `game` | `sakura.proxy.rlwy.net` |

Skip `-p` with `~/.ssh/config`:

```
Host game.gotype.fun
  Port 58372
  User play
```

Then: `ssh game.gotype.fun`

## Install

```bash
go install github.com/kjkusap/monkeytype-clone/cmd/gotype@latest
```

Or from this repo:

```bash
go run ./cmd/gotype
```

## Usage

```
gotype
```

1. Pick **time**, **words**, or **quote** mode (`t` / `w` / `o`, or ↑↓)
2. Pick duration / word count / quote length (tab to value, ↑↓ to change)
3. Enter to start — type the prompt
4. See WPM chart, accuracy, and breakdown on finish

### Keys

| Key | Action |
|-----|--------|
| ↑↓ / j k | Change mode or value |
| tab | Switch focus / restart test |
| t / w / o | Time / words / quote mode |
| p | Cycle theme |
| v | Toggle roast ↔ stoic voice |
| n | Toggle ninja caret (smooth + trail) |
| enter / space | Start (menu) or next test (results) |
| esc | Back to menu |
| q / ctrl+c | Quit |

## Modes

- **time** — 15 / 30 / 60 / 120 seconds
- **words** — 10 / 25 / 50 / 100 words
- **quote** — stoic/philosophy quote races · short / medium / long (finish the passage)

Metrics: WPM from correct chars÷5÷minutes, accuracy from correct÷typed.

## Multiplayer (SSH)

Over `gotype-ssh`, press **m** on the menu:

1. **c** create room → share 4-letter code
2. Friend **j** join → type code
3. Host **s** / enter to start (need 2–4 players)
4. 3s countdown → same prompt → live WPM bars + pace ghost
5. Podium — **best of 3** series score
6. **enter / tab / r** next race (same room); after match win, resets series
7. New friends: **j** + same code during podium/lobby
8. Chat: **g** = gg, **/** compose (`glhf`, `wp`, …)
9. **esc** leave room

👑 = win streak. Solo: finish a run to unlock **pace ghost** on the next test.

Themes: **p** on main menu cycles amber / amber light / nord / olive / dracula.

### Roast

After a solo run, results show a short roast of your WPM/acc.

- No API key → canned local roasts
- **Google Gemini**: set `GEMINI_API_KEY` (or `GOOGLE_API_KEY`). Default model `gemini-2.5-flash-lite`. Override with `ROAST_MODEL`.
- **OpenAI-compatible**: set `ROAST_API_KEY` / `OPENAI_API_KEY`. Optional `ROAST_BASE_URL`, `ROAST_MODEL` (default `gpt-4o-mini`).
- Force backend: `ROAST_PROVIDER=google` or `openai`
- Menu **v** toggles result voice: **roast** (default) ↔ **stoic** (calm mentor lines)

### Lightning tips

On results, press **t** (when configured) → pick sats → QR + bolt11 invoice (LNURL-pay).

```bash
export TIP_LIGHTNING_ADDRESS="you@getalby.com"
# or
export TIP_LNURL="https://…/.well-known/lnurlp/…"
```

## Play over SSH

`gotype-ssh` serves the same TUI over SSH (Charm Wish). Auth is open — any user/password.

### Local

```bash
go run ./cmd/gotype-ssh
# default listen 0.0.0.0:2222

ssh play@localhost -p 2222
# any username works; password can be empty/anything
```

Optional: set `PORT` (HTTP landing, default `8080`), `SSH_PORT` (default `2222`), and/or `SSH_HOST_KEY` (PEM) so the host key stays stable across restarts.

```bash
export SSH_HOST_KEY="$(cat ./host_ed25519)"
PORT=8080 SSH_PORT=2222 go run ./cmd/gotype-ssh
```

### Railway

Deploy from this repo (Dockerfile builds `gotype-ssh`). HTTP landing on **8080**, SSH on **2222** (TCP Proxy). Set secret `SSH_HOST_KEY` to a durable ed25519 PEM.

Custom domain **gotype.fun** → HTTP landing. SSH host **game.gotype.fun** (CNAME → TCP proxy):

```bash
ssh play@game.gotype.fun -p 58372
```
