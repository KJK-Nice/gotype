# PRD: gotype progression + Lightning shop

**Status:** decision pack (ready for a later implement effort)  
**Map:** [Gamify gotype: progression + Lightning shop PRD](https://github.com/KJK-Nice/gotype/issues/2)  
**Glossary:** [`CONTEXT.md`](../../CONTEXT.md)

## 1. Goal

Ship a design for **race-game progression** (Inventory, Cosmetics, Consumables, Season Pass / XP) and a **non-custodial Lightning economy** at equal depth: **Tip + Buy** only. No in-TUI wallet, no stake pots, no cash-out.

Existing today: Tip via LNURL-pay → bolt11 QR (`internal/ln`). Multi races with ephemeral names. This PRD adds persisted Player progression and a sats shop.

## 2. Non-goals (v1)

- Custodial / in-TUI Lightning wallet or seed storage
- Soft shop **credits** / currency ledger (sats Buy per SKU only; XP is Season-only)
- Stake / prize pots / wagering
- Earn-and-cash-out sats
- P2P item trading or marketplace
- OAuth / web login

## 3. Player identity

- Persisted **Player**: unique display name (3–16 `[a-zA-Z0-9_]`, case-insensitive) + **Claim Code**.
- **Claim Code**: 12-char Crockford base32, shown as `XXXX-XXXX-XXXX`; store password hash only (argon2id/bcrypt).
- **No recovery v1** — lose code → new Player; warn at register.
- **Rename**: once per Season; old name freed.
- **Sessions**: re-enter name + Claim Code each SSH; **single active** via `Player.activeSessionID` (new claim kicks older).
- **When**: optional until shop / Season Pass / Inventory equip; guests may race without claim.
- **Abuse**: rate-limit register + claim (per IP + per name).

## 4. Progression — Season Pass + XP

| Parameter | Value |
|---|---|
| Season length | 60 days |
| Tracks | 20 free + 20 premium (shared XP thresholds) |
| XP solo finish | 10 |
| XP multi finish | 25 |
| Incomplete / spectate | 0 XP |
| Daily soft cap | 200 XP / UTC day (extra races grant 0 Season XP) |
| Curve | Linear 100 XP / tier → 2000 XP to tier 20 |
| Season premium Buy | 2100 sats (current Season only) |

Premium track requires Buy. Free track rewards include **Matrix** (Theme) at tier 10. Premium includes **Make it Rain** (FX) at tier 15. Other tier Cosmetics: placeholder names (deferred).

### Season rollover

Keep Player, Inventory, Equipment, Order history. New **SeasonProgress** at 0 XP; reset rename-used; Cosmetics stay equipped. Season 2 *content* naming deferred.

## 5. Inventory, Cosmetics, Consumables

### Equip slots (one each)

Theme · Caret · Title · **FX**

### Named Cosmetics (v1)

| SKU | Slot | Source |
|---|---|---|
| Matrix | Theme | Free track tier 10 |
| Make it Rain | FX (freefall glyphs) | Premium track tier 15 |

### Consumable classes

| Class | Effect | Where |
|---|---|---|
| Reveal | Peek next words | Solo + casual multi |
| Calm | Reduce error shake/flash once | Solo + casual multi |
| Retry | Free restart | **Solo only** |
| Heart | +1 HP (max 5) | **hardcore (Three-Strike) only** |

- **Banned** in match-point / ranked series.
- **Per race:** at most one use **per class**.
- Banned effects: WPM cheats, time freeze, opponent sabotage.

### Shop prices (sats)

| SKU | Sats |
|---|---|
| Reveal | 21 |
| Calm | 21 |
| Retry | 50 |
| Heart | 100 |
| Season premium | 2100 |

Shop sells **Consumables + Season premium only**. Named Cosmetics come from Season tracks.

## 6. hardcore (Three-Strike)

- Domain: **Three-Strike**. Lobby label: **hardcore**.
- Host lobby toggle: `classic` ↔ `hardcore` before countdown; locked when countdown starts.
- Start **3 HP**; each incorrect character commit −1; no backspace refund; 0 HP = DNF, no Season XP.
- Lobby copy: `hardcore · 3 HP · typo −1`. Race HUD: `❤❤❤`.

## 7. Lightning economy

### Tip (existing)

Operator LNURL-pay / lightning address → bolt11 QR. Not an Order. Keep separate from shop accounting (ops split Tip destination vs shop node — deferred runbook).

### Buy (new)

- Non-custodial: server creates inbound invoices; players pay from their own wallets.
- **Default settle stack: LNBits** — REST create inbound invoice (`out: false`), correlate Order via `extra` / external id, per-invoice webhook as hint, **poll to confirm paid before grant**.
- Alby Hub alternate: NWC/NIP-47 (not deprecated Alby Wallet HTTP API).
- Research: `docs/research/lnbits-alby-shop-invoice-settle.md` @ branch `research/lnbits-alby-shop-settle`.

### Order lifecycle

States: `created` → `invoiced` → `paid` → `granted`; also `expired` | `failed`.

- One SKU (or Season premium) per Order; no cart; no credits.
- Invoice TTL **15 minutes**; late pay still grants if `payment_hash` matches.
- Store `payment_hash` + LNBits `checking_id`; grant **once** (idempotent).
- Consumable Buy → stack qty +1; Season premium → reject if already owned this Season.
- Claimed Player required.
- TUI: Tip-like full-screen QR wait + poll; esc leaves wait (invoice may still settle → grant).

## 8. TUI information architecture

- **Progression hub** with tabs: inventory · shop · pass · equip.
- **Global hotkeys** `i` / `s` / `p` / `e` open the same surfaces as overlays from menu/results.
- Buy wait = blocking full-screen.
- Prototype (throwaway): `go run ./cmd/proto-shopia` on `prototype/tui-ia-shop`.

## 9. Persistence (domain entities)

Not SQL — implementer chooses store.

| Entity | Role |
|---|---|
| Player | Identity, claim hash, rename flag, activeSessionID |
| InventoryItem | Player + SKU + qty/owned |
| Equipment | Player + slot → SKU |
| Season | Window + track definition ref |
| SeasonProgress | XP, premiumUnlocked, claimed tiers |
| Order | Buy state machine + invoice correlation |
| DailyXP | UTC day + xp granted toward cap |

Catalog SKUs = config/code, not DB rows.

## 10. Open / deferred

- Remaining Season track Cosmetic names (non-Matrix / non-Rain).
- Season 2 track content.
- Ops: Tip LNURL vs shop LNBits accounting runbook.
- Anti-cheat beyond daily XP cap.
- Optional: Lightning address as Claim Code recovery.

## 11. Traceability

Decisions recorded on map [#2](https://github.com/KJK-Nice/gotype/issues/2) and closed tickets #3–#11.
