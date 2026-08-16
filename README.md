# gotype

**Monkeytype energy, SSH chaos, terminal aesthetic.**

Race your friends (or your ego) in a Bubble Tea TUI — solo time trials, multiplayer lobbies, stoic, Naval, and absurdist quotes, AI roasts, and optional Lightning tips. No browser tab. Just you, the prompt, and whoever else showed up on port 58372.

## Play now

**Website:** [gotype.fun](https://gotype.fun)

**SSH (the real experience):**

```bash
ssh play@game.gotype.fun -p 58372
```

Any username works. Password can be empty — we're not gatekeeping the vibes.

```ssh-config
# ~/.ssh/config — skip -p every time
Host game.gotype.fun
  Port 58372
  User play
```

Then: `ssh game.gotype.fun`

---

## Why this exists

- **Terminal-native** — WPM charts, ninja caret, themes, pace ghost. Feels like home if you live in `vim`/`tmux`.
- **Multiplayer over SSH** — create a 4-letter room, trash-talk with `gg`, best-of-3 series, live race bars.
- **Quote mode** — Marcus Aurelius, Seneca, **Naval Ravikant**, Paul Graham, and absurdist one-liners. Finish the passage or eat the L.
- **Roast or stoic** — Gemini/OpenAI roasts your run, or a calm mentor line if you're fragile today.
- **Progression** — login with a Lightning wallet, earn XP, Season Pass tiers, cosmetics, consumables, hardcore (Three-Strike) mode.
- **Lightning** — tip the operator or buy shop items with sats (Phoenixd on Railway).

Built with Go + [Bubble Tea v2](https://github.com/charmbracelet/bubbletea). Deployed on [Railway](https://railway.com). Architecture in [ARCHITECTURE.md](./ARCHITECTURE.md).

## Quick start (local)

```bash
go run ./cmd/gotype          # solo TUI
go run ./cmd/gotype-ssh      # SSH server (port 2222)
```

Or install solo binary:

```bash
go install github.com/KJK-Nice/gotype/cmd/gotype@latest
gotype
```

## Controls

1. Pick **time**, **words**, **quote**, or **ai** (`t` / `w` / `o` / `a` when LLM configured, or `←/→` when mode is focused)
2. Pick duration / word count / quote length (`↑/↓` to focus value row, `←/→` to change)
3. Enter or space to start — type the prompt
4. Results: WPM chart, accuracy, roast/stoic line

| Key | Action |
|-----|--------|
| ↑↓ / j k | Switch focus between mode and value |
| ←→ | Change mode or value (by focus) |
| tab | Toggle mode ↔ value focus |
| t / w / o / a | Time / words / quote / ai mode (`a` needs LLM key) |
| u | Cycle theme |
| v | Toggle roast ↔ stoic voice |
| n | Toggle ninja caret |
| y | Toggle daily words |
| g | Toggle pace ghost |
| l | Login with Lightning wallet |
| i / s / p / e | Inventory / shop / pass / equip |
| m | Multiplayer (SSH) |
| enter / space | Start or next test |
| esc | Back to menu |
| q / ctrl+c | Quit |

## Modes

| Mode | What |
|------|------|
| **time** | 15 / 30 / 60 / 120 seconds |
| **words** | 10 / 25 / 50 / 100 words |
| **quote** | Short / medium / long passages — stoics, Naval, absurdist fake-wisdom |
| **ai** | LLM stoic/Naval-ish passage (needs API key; solo only) |

WPM = correct chars ÷ 5 ÷ minutes. Accuracy = correct ÷ typed.

## Multiplayer (SSH)

Press **m** on the menu:

1. **c** create room → share 4-letter code
2. Friend **j** join
3. Host **s** / enter to start (2–4 players)
4. 3-2-1 countdown → same prompt → live WPM bars
5. Best-of-3 series — 👑 = win streak
6. **g** = gg, **/** chat (`glhf`, `wp`, …)
7. **esc** leave room

Enable **hardcore** (Three-Strike) in lobby — 3 HP, typos hurt, Heart consumables exist.

## Progression & shop (SSH)

Login with a **Lightning wallet** (anonymous LNURL-auth — Wallet of Satoshi, Phoenix, Zeus, Breez, Alby, or Blink). Earn XP from races, unlock Season Pass tiers, equip cosmetics (themes, carets, FX), buy consumables with sats.

Tips and shop checkout use **Lightning** (BOLT11 QR) when Phoenixd is configured on the server.

## Roast

After solo runs: short roast of your WPM/acc.

- No API key → local canned roasts
- **Gemini:** `GEMINI_API_KEY` or `GOOGLE_API_KEY` (`ROAST_MODEL`, default `gemini-3.1-flash-lite`)
- **OpenAI-compatible:** `ROAST_API_KEY` / `OPENAI_API_KEY`, optional `ROAST_BASE_URL`
- **v** on menu toggles roast ↔ stoic

## Deploy your own

`gotype-ssh` = Wish SSH server + HTTP landing. See [ARCHITECTURE.md](./ARCHITECTURE.md) for Redis, Phoenixd, env vars.

Railway: Dockerfile builds `gotype-ssh`. HTTP **8080**, SSH **2222** (TCP proxy). Set `SSH_HOST_KEY` (ed25519 PEM), `REDIS_URL`, `PHOENIXD_URL` + `PHOENIXD_PASSWORD` for production.

DNS for this deployment:

| Type | Name | Content |
|------|------|---------|
| CNAME | `game` | `sakura.proxy.rlwy.net` |

---

**Play:** [gotype.fun](https://gotype.fun) · **SSH:** `play@game.gotype.fun -p 58372`

Made for people who think typing fast in a terminal is a personality trait.
