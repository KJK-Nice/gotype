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
