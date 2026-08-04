## Agent skills

### Issue tracker

GitHub Issues via `gh` (`KJK-Nice/monkeytype-clone`). See `docs/agents/issue-tracker.md`.

### Triage labels

Defaults: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context — root `CONTEXT.md` + `docs/adr/`. See `docs/agents/domain.md`.

## Cursor Cloud specific instructions

Pure Go module (`go.mod` pins `go 1.26.1`). `GOTOOLCHAIN=auto` is set, so plain `go` commands auto-fetch the pinned toolchain — no manual Go install needed. Standard commands: build `go build ./...`, lint `go vet ./...`, test `go test ./...`.

### Services

Two binaries (see `ARCHITECTURE.md`):

- `cmd/gotype` — solo TUI. Run inside a real terminal/PTY (`go run ./cmd/gotype`); it will paint nothing if stdout is piped/non-tty.
- `cmd/gotype-ssh` — production server: Wish SSH on `SSH_PORT` (default `2222`) + HTTP landing on `PORT` (default `8080`, `/` and `/health`). Run with `go run ./cmd/gotype-ssh`. Connect with `ssh -p 2222 <anyuser>@localhost` (password auth accepts anything).

### Persistence

No Redis needed for local dev — persistence falls back to a JSON file at `$GOTYPE_DATA_DIR/data.json` (OS temp dir if `GOTYPE_DATA_DIR` unset). `REDIS_URL`, `PHOENIXD_*`, and roast API keys are production-only and optional; the app runs fully without them (roasts use local canned lines).

### Known time-dependent test failure

`TestGrantFinishAndMatrixUnlock` in `internal/progress` is a pre-existing, wall-clock-dependent test — not an environment problem. It hardcodes simulated dates (`2026-07-31`..`2026-08-04`), while `internal/persist` seeds a season window from real `time.Now()` on store open. When today's date falls inside that simulated range, two overlapping seasons exist and Go's randomized map iteration splits XP across them, so tier 10 (Matrix reward) is never reached. All other packages pass.
