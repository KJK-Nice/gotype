# Research: LNBits / Alby Hub shop invoice create + settle

**Issue:** [#3 — Research LNBits/Alby Hub shop invoice settle](https://github.com/KJK-Nice/monkeytype-clone/issues/3)  
**Question:** For a non-custodial Buy flow: how do we create an inbound Lightning invoice on LNBits and/or Alby Hub, correlate it to an Order id, and reliably detect settlement (webhook and/or poll) so Inventory can be granted — citing primary docs/APIs only?  
**Audience:** PRD / decision pack (not implementation).  
**Researched:** 2026-07-31  
**Sources:** Official LNBits docs + LNBits source; Alby Hub README/source; NIP-47; Alby developer guides (NWC). The deprecated Alby Wallet HTTP API is noted only to avoid confusion with Alby Hub.

---

## Executive recommendation (for PRD)

**Default provider for the gotype shop Buy flow: LNBits.**

Evidence supports LNBits as the lower-friction fit for a Go server that creates invoices and verifies payment without holding a custodial wallet in the TUI:

| Need | LNBits | Alby Hub |
| --- | --- | --- |
| Create inbound invoice | Documented REST `POST /api/v1/payments` with `out: false` ([Payments API](https://docs.lnbits.com/api/core/payments)) | NIP-47 `make_invoice` over NWC ([NIP-47](https://nips.nostr.com/47), [Hub README](https://github.com/getAlby/hub/blob/master/README.md)) |
| Correlate Order id | Documented `extra` object; source also accepts `external_id` (≤256, no spaces/newlines) | NIP-47 `metadata` on `make_invoice` (≤4096 chars), echoed on lookup/notifications |
| Settlement push | Per-invoice HTTP `webhook` POST of Payment JSON ([concepts](https://docs.lnbits.com/guide/concepts), [source `dispatch_webhook`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/notifications.py)); optional WebSocket | NIP-47 `payment_received` notifications (Nostr), not an HTTP webhook ([NIP-47](https://nips.nostr.com/47), [Hub README](https://github.com/getAlby/hub/blob/master/README.md)) |
| Settlement poll | `GET /api/v1/payments/{checking_id}` → `paid` + `details.status` ([Payments API](https://docs.lnbits.com/api/core/payments)) | NIP-47 `lookup_invoice` by `payment_hash` / `invoice` ([NIP-47](https://nips.nostr.com/47)) |
| Auth for receive-only server | Invoice/read key via `X-Api-Key` ([Authentication](https://docs.lnbits.com/guide/authentication), [concepts](https://docs.lnbits.com/guide/concepts)) | NWC connection secret with `make_invoice` + `lookup_invoice` (+ `notifications` / `payment_received`) ([Hub README](https://github.com/getAlby/hub/blob/master/README.md), [NWC client guide](https://guides.getalby.com/developer-guide/developer-guide/bitcoin-payments-agent-skill/references/nwc-client/nwc-client.md)) |
| Go ergonomics | Plain HTTPS JSON | Must speak NWC (Nostr + NIP-44); Hub’s own HTTP API is explicitly **not a public spec** — apps should use NIP-47 ([Hub README](https://github.com/getAlby/hub/blob/master/README.md)) |

**Alby Hub remains a supported alternate** when the operator already runs Hub and prefers NWC-scoped app connections / isolated sub-wallets. Do **not** plan the shop around the deprecated Alby Wallet HTTP API (`https://api.getalby.com/...`) — that is a different product surface marked deprecated in Alby’s guides ([Invoices](https://guides.getalby.com/developer-guide/developer-guide/alby-wallet-api/reference/api-reference/invoices), [Webhooks](https://guides.getalby.com/developer-guide/developer-guide/alby-wallet-api/reference/api-reference/webhook-endpoints)).

**Reliable grant pattern (either provider):** treat push (webhook / NWC notification) as a hint; **always confirm paid status via poll** before granting Inventory; store `payment_hash` (and LNBits `checking_id`) on the Order; make grant idempotent.

---

## Scope note: Tips vs Shop

Tips already use LNURL-pay / bolt11 to an operator address and are **not** shop Buys ([CONTEXT.md](../../CONTEXT.md)). Shop Orders are server-tracked Buy intents correlated to a Lightning invoice until paid or expired. Neither LNBits nor Alby Hub needs to become a wallet inside the SSH TUI — the **server** creates invoices and verifies settlement.

---

## 1. LNBits

### 1.1 Create inbound invoice

**Endpoint:** `POST /api/v1/payments`  
**Auth:** Invoice key or Admin key (`X-Api-Key`)  
**Docs:** [Payments API — Create Invoice](https://docs.lnbits.com/api/core/payments), [API Reference quick start](https://docs.lnbits.com/api/)

Required / primary body fields (official docs):

| Field | Required | Meaning |
| --- | --- | --- |
| `out` | Yes | Must be `false` for receive |
| `amount` | Yes | Amount in satoshis (docs) |
| `memo` | No | Invoice description (visible to payer) |
| `webhook` | No | URL called when payment is received |
| `expiry` | No | Seconds until expiry (docs default 3600; FAQ also notes instance default can be 24h) |
| `extra` | No | Custom metadata attached to the payment |

Example from official docs:

```bash
curl -X POST https://your-lnbits.com/api/v1/payments \
  -H "X-Api-Key: YOUR_INVOICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"out": false, "amount": 100, "memo": "test"}'
```

**Response 201 (docs):** `payment_hash`, `payment_request` (BOLT11), `checking_id` (for status polling).

**Source model (CreateInvoice)** also includes fields not all listed on the Payments API page: `unit` (default `"sat"`), `extra`, `webhook`, `expiry`, `labels`, `external_id` (max length 256), hold-invoice `payment_hash`, etc.  
Source: [`lnbits/core/models/payments.py` — `CreateInvoice`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/models/payments.py).  
Create path wires `webhook`, `extra`, `labels`, `external_id` into invoice creation: [`create_invoice` / `create_payment_request`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/payments.py).

Instance OpenAPI always matches the running version at `https://your-lnbits.com/docs` ([API Reference](https://docs.lnbits.com/api/)).

### 1.2 Correlate Order id

Three first-party mechanisms, in preference order for a merchant Order:

1. **`extra` (documented)** — arbitrary JSON object stored on the payment; list responses show e.g. `"extra": {"order_id": "123"}` ([Payments API](https://docs.lnbits.com/api/core/payments), [Core Concepts](https://docs.lnbits.com/guide/concepts), [Models](https://docs.lnbits.com/dev/models)). Extension docs use `extra.tag` / `extra.item_id` when filtering paid invoices ([Background Tasks](https://docs.lnbits.com/dev/tasks)).
2. **`external_id` (source API model)** — first-class column/filter on payments; validated ≤256 chars, no spaces/newlines ([`is_valid_external_id`](https://github.com/lnbits/lnbits/blob/main/lnbits/helpers.py), [`CreateInvoice.external_id`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/models/payments.py), filterable via `PaymentFilters.external_id`). **Not currently called out on the public Payments API parameter table** — verify against instance `/docs` before relying on it in the PRD as a documented contract.
3. **`memo`** — human-readable; visible on the BOLT11 description. Fine for display; weak as sole correlation (length limits, payer-visible, not a structured key). Docs: memo is “Invoice description (visible to payer)” ([Payments API](https://docs.lnbits.com/api/core/payments)).

**Recommended correlation for PRD:** store Order id in the game DB keyed by `payment_hash` / `checking_id` returned at create time; also put Order id in `extra` (e.g. `{"order_id":"<uuid>"}`) so webhook/list payloads can recover it if needed. Optionally set `external_id` to the same Order id after confirming the running LNBits version exposes it.

### 1.3 Detect settlement

#### A. Per-invoice HTTP webhook (push)

- Set `webhook` on create ([Payments API](https://docs.lnbits.com/api/core/payments)).
- Core concepts: `webhook` is “optional URL called when payment status changes” ([Core Concepts](https://docs.lnbits.com/guide/concepts)).
- Background task: “Webhook processor — On event — Calls webhook URLs for paid invoices” ([Background Tasks](https://docs.lnbits.com/dev/tasks)).
- Implementation: `dispatch_webhook` POSTs `json=payment.json()` to the configured URL, 40s timeout, records HTTP status on `webhook_status`; invalid callback URLs can be rejected by admin `lnbits_callback_url_rules` ([`notifications.py`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/notifications.py), [`check_callback_url`](https://github.com/lnbits/lnbits/blob/main/lnbits/helpers.py)).
- Payment model includes `webhook`, `webhook_status`, `extra`, `payment_hash`, `checking_id`, `status` (`pending` / `success` / `failed`) ([Models](https://docs.lnbits.com/dev/models), [source Payment](https://github.com/lnbits/lnbits/blob/main/lnbits/core/models/payments.py)).

**Gotcha:** Official docs do **not** document a signed webhook HMAC. Treat the POST as untrusted; include a high-entropy secret in the webhook URL path/query and/or re-check with `GET /api/v1/payments/{checking_id}` before granting Inventory.

#### B. Poll status (authoritative confirm)

`GET /api/v1/payments/{checking_id}` with Invoice or Admin key ([Payments API](https://docs.lnbits.com/api/core/payments)):

- Response: `paid` (boolean), `details.status` ∈ `"pending"` | `"success"` | `"failed"`.
- Grant Inventory only when `paid == true` / `status == "success"`.

Also: list/filter payments (`GET /api/v1/payments`, paginated variants) including by `status` ([Payments API](https://docs.lnbits.com/api/core/payments), [API filtering](https://docs.lnbits.com/api/)).

#### C. WebSocket (optional push)

`WS /api/v1/ws/{wallet_id}` — real-time payment updates; wallet id in path; docs say no API key required for public payment notifications ([WebSockets API](https://docs.lnbits.com/api/core/websockets), [WebSockets guide](https://docs.lnbits.com/guide/core/websockets)). Useful for live UX; still confirm via poll before Inventory grant. **Security note:** anyone who knows `wallet_id` can subscribe — do not treat WS as a secret channel.

### 1.4 Auth / API keys

| Key | Create invoice | Check status | Send payments |
| --- | --- | --- | --- |
| Invoice / read (`inkey`) | Yes | Yes | No |
| Admin (`adminkey`) | Yes | Yes | Yes |

Sources: [Core Concepts — API Keys](https://docs.lnbits.com/guide/concepts), [Wallets & Accounts](https://docs.lnbits.com/guide/core/wallets-and-accounts), [Authentication](https://docs.lnbits.com/guide/authentication).

Header (recommended): `X-Api-Key: YOUR_KEY` ([Authentication](https://docs.lnbits.com/guide/authentication)).  
Shop server should use the **Invoice key only** unless refunds/payouts are in scope.

Rate limit default: 200 requests/minute/IP (`LNBITS_RATE_LIMIT_NO` / `LNBITS_RATE_LIMIT_UNIT`) ([API Reference](https://docs.lnbits.com/api/)).

### 1.5 Go server gotchas (LNBits)

- Use plain `net/http` or any REST client — no Nostr stack required.
- Persist `checking_id` and `payment_hash` on Order at create time; poll by `checking_id` as documented.
- **Units:** create `amount` is in sats (docs); Payment model documents `amount` in **millisatoshis** ([Models](https://docs.lnbits.com/dev/models), [Core Concepts](https://docs.lnbits.com/guide/concepts)). The Check Payment Status table says `details.amount` in satoshis ([Payments API](https://docs.lnbits.com/api/core/payments)) — **verify against instance OpenAPI** before asserting units in code.
- Webhook URL must be reachable by the LNBits instance; may be constrained by `lnbits_callback_url_rules` ([source](https://github.com/lnbits/lnbits/blob/main/lnbits/helpers.py)).
- Webhook is fire-and-forget with status recorded; failed deliveries are logged / status-coded, not a strong delivery guarantee — keep a poller/reconciler ([`dispatch_webhook`](https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/notifications.py), payment checker every 30 minutes for pending payments ([Background Tasks](https://docs.lnbits.com/dev/tasks))).
- Expiry: set explicitly for shop Orders (`expiry` seconds) ([Payments API](https://docs.lnbits.com/api/core/payments), [Payments FAQ](https://docs.lnbits.com/guide/faq/payments)).
- Hold invoices exist (settle/cancel APIs) but are optional escrow complexity — not required for a standard Buy → grant Inventory flow ([Payments API — Hold Invoices](https://docs.lnbits.com/api/core/payments)).

---

## 2. Alby Hub

### 2.1 Integration surface: NWC (NIP-47), not Hub HTTP

Alby Hub is NWC-first. It supports `make_invoice` and `lookup_invoice` (and notifications via the NIP-47 notifier path) ([Hub README — NIP-47 Supported Methods](https://github.com/getAlby/hub/blob/master/README.md)).

Hub’s internal HTTP API “is not a public spec”; **“apps are recommended to use NIP-47 where possible”** ([Hub README — Frontend / HTTP](https://github.com/getAlby/hub/blob/master/README.md)).

Auth credential: a `nostr+walletconnect://...` connection secret (treat like an API key) ([NWC client guide](https://guides.getalby.com/developer-guide/developer-guide/bitcoin-payments-agent-skill/references/nwc-client/nwc-client.md)). Create app connections in Hub (`/apps/new`) with scoped `request_methods` and `notification_types` (e.g. `payment_received`) ([Hub README](https://github.com/getAlby/hub/blob/master/README.md)).

### 2.2 Create inbound invoice

NIP-47 method `make_invoice` ([NIP-47](https://nips.nostr.com/47)):

```json
{
  "method": "make_invoice",
  "params": {
    "amount": 123,
    "description": "string",
    "description_hash": "string",
    "expiry": 213,
    "metadata": {}
  }
}
```

- `amount` is in **millisatoshis** ([NIP-47](https://nips.nostr.com/47); Alby guide: “All referenced files … operate in millisats” ([NWC client](https://guides.getalby.com/developer-guide/developer-guide/bitcoin-payments-agent-skill/references/nwc-client/nwc-client.md))).
- Response includes `invoice` (BOLT11), `payment_hash`, `amount`, optional `metadata`, etc. ([NIP-47](https://nips.nostr.com/47)).

Alby Hub controller accepts `metadata` on `make_invoice` and passes it to the transactions service ([`make_invoice_controller.go`](https://github.com/getAlby/hub/blob/master/nip47/controllers/make_invoice_controller.go)). Hub stores app-provided metadata on transactions ([Hub README — Transactions Service](https://github.com/getAlby/hub/blob/master/README.md)). Amount must be a whole number of satoshis (≥ 1 sat) in Hub’s transactions service ([`transactions_service.go`](https://github.com/getAlby/hub/blob/master/transactions/transactions_service.go)).

Official JS path for backends: `NWCClient` from `@getalby/sdk` ([NWC JS SDK](https://guides.getalby.com/developer-guide/nostr-wallet-connect-api/building-lightning-apps/nwc-js-sdk)).

### 2.3 Correlate Order id

Use NIP-47 **`metadata`** on `make_invoice` (optional object). Spec: metadata MAY be stored alongside invoices; **MUST be ≤ 4096 characters** or dropped ([NIP-47 — Metadata](https://nips.nostr.com/47)).

Returned again on `lookup_invoice` / `list_transactions` / `payment_received` notification payloads (`metadata` field) ([NIP-47](https://nips.nostr.com/47)).

Practical pattern: `metadata: { "order_id": "<uuid>" }` (plus keep Order ↔ `payment_hash` in the game DB).

`description` is BOLT11-visible (like LNBits `memo`) — secondary correlation only.

### 2.4 Detect settlement

#### A. NWC notifications (push)

If the wallet service supports notifications, info event lists types such as `payment_received` ([NIP-47](https://nips.nostr.com/47)).  
`payment_received` payload includes `payment_hash`, `preimage`, `settled_at`, `metadata`, `state` (optional, e.g. `"settled"`) ([NIP-47 — payment_received](https://nips.nostr.com/47)).

Hub: app connect URL can request `notification_types=payment_received`; Hub publishes NWC notification events when payments are received ([Hub README](https://github.com/getAlby/hub/blob/master/README.md); notification types in [`nip47/notifications`](https://github.com/getAlby/hub/blob/master/nip47/notifications/models.go)).

**This is not an HTTP webhook.** The Go server must maintain a NWC notification subscription over the configured Nostr relay.

#### B. Poll with `lookup_invoice`

```json
{
  "method": "lookup_invoice",
  "params": {
    "payment_hash": "31afdf1..",
    "invoice": "lnbc50n1..."
  }
}
```

One of `payment_hash` or `invoice` is required. Settled invoices include `settled_at` / `state: "settled"` ([NIP-47](https://nips.nostr.com/47)).  
Hub implements lookup; note Hub caveat: **`NOT_FOUND` error code not supported** ([Hub README](https://github.com/getAlby/hub/blob/master/README.md)).

### 2.5 Auth / keys

- Store one server-side NWC URL for the shop app connection ([NWC client guide](https://guides.getalby.com/developer-guide/developer-guide/bitcoin-payments-agent-skill/references/nwc-client/nwc-client.md)).
- Scope methods to receive + verify: at minimum `make_invoice`, `lookup_invoice`; add notifications `payment_received` for push ([Hub README query params](https://github.com/getAlby/hub/blob/master/README.md)).
- Prefer **not** granting `pay_invoice` to the shop connection unless payouts are required.
- Optional `isolated=true` sub-wallet connections exist but Hub docs say not to combine with custom request methods / notification types / budget when using isolated ([Hub README](https://github.com/getAlby/hub/blob/master/README.md)) — evaluate carefully for shop accounting.

### 2.6 Go server gotchas (Alby Hub)

- Need a NWC client stack (Nostr relay WebSocket, NIP-44 encryption, request/response event kinds). Alby documents JS `@getalby/sdk` for backends ([NWC JS SDK](https://guides.getalby.com/developer-guide/nostr-wallet-connect-api/building-lightning-apps/nwc-js-sdk)); community Go clients are listed in [awesome-nwc](https://github.com/getAlby/awesome-nwc) (not first-party Alby Go SDK).
- Amounts in **msats** on the wire ([NIP-47](https://nips.nostr.com/47)).
- No documented per-invoice HTTPS webhook on Hub’s public integration path — plan notification + poll.
- Do not build against Hub’s JWT HTTP API as a stable public contract ([Hub README](https://github.com/getAlby/hub/blob/master/README.md)).
- Do not confuse with deprecated Alby Wallet API webhooks (`POST https://api.getalby.com/webhook_endpoints`, SVIX) ([deprecated Webhook Endpoints](https://guides.getalby.com/developer-guide/developer-guide/alby-wallet-api/reference/api-reference/webhook-endpoints)).

---

## 3. Suggested Order ↔ settle flow (provider-agnostic)

```text
Player Buy
  → Server creates Order (pending)
  → Provider: create invoice (amount, expiry, order correlation metadata)
  → Persist payment_hash (+ LNBits checking_id), bolt11 on Order
  → Show bolt11 / QR in TUI (non-custodial; no wallet keys in client)
  → On webhook/WS/NWC notification OR poll tick:
       confirm paid via provider status API
       if paid and Order still pending → grant Inventory (idempotent)
  → On expiry without pay → Order expired (no grant)
```

---

## 4. Comparison snapshot

| Dimension | LNBits | Alby Hub (NWC) |
| --- | --- | --- |
| Protocol | HTTPS REST (+ optional WS) | NIP-47 over Nostr relay |
| Create | `POST /api/v1/payments` `out:false` | `make_invoice` |
| Order correlation | `extra` (docs); `external_id` (source) | `metadata` (spec ≤4096 chars) |
| Push settle | Per-invoice HTTP webhook; WS | `payment_received` notification |
| Poll settle | `GET .../payments/{checking_id}` | `lookup_invoice` |
| Creds | Invoice API key | NWC connection secret |
| Public HTTP merchant API | Yes | No (use NWC) |
| Fit for Go shop server | Strong default | Strong if Hub/NWC already chosen |

---

## 5. Open points for PRD (not blocking the default)

1. Confirm target LNBits version exposes `external_id` on create in `/docs` before treating it as the primary Order key.
2. Reconcile amount units (sats vs msats) against the deployed LNBits OpenAPI for webhook/poll parsers.
3. If operator standardizes on Alby Hub only, flip default to NWC and budget for notification subscription + Go NWC client maintenance.
4. Tips remain on existing LNURL-pay path; shop should use a dedicated LNBits wallet (or Hub app connection) separate from tip destination for accounting clarity ([Wallets best practices](https://docs.lnbits.com/guide/core/wallets-and-accounts)).

---

## Primary sources

### LNBits
- https://docs.lnbits.com/api/
- https://docs.lnbits.com/api/core/payments
- https://docs.lnbits.com/api/core/websockets
- https://docs.lnbits.com/guide/concepts
- https://docs.lnbits.com/guide/authentication
- https://docs.lnbits.com/guide/core/wallets-and-accounts
- https://docs.lnbits.com/guide/core/websockets
- https://docs.lnbits.com/guide/faq/payments
- https://docs.lnbits.com/dev/models
- https://docs.lnbits.com/dev/tasks
- https://github.com/lnbits/lnbits/blob/main/lnbits/core/models/payments.py
- https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/payments.py
- https://github.com/lnbits/lnbits/blob/main/lnbits/core/services/notifications.py
- https://github.com/lnbits/lnbits/blob/main/lnbits/helpers.py

### Alby Hub / NWC
- https://github.com/getAlby/hub/blob/master/README.md
- https://github.com/getAlby/hub/blob/master/nip47/controllers/make_invoice_controller.go
- https://github.com/getAlby/hub/blob/master/nip47/controllers/lookup_invoice_controller.go
- https://github.com/getAlby/hub/blob/master/nip47/notifications/models.go
- https://github.com/getAlby/hub/blob/master/transactions/transactions_service.go
- https://nips.nostr.com/47
- https://guides.getalby.com/developer-guide/nostr-wallet-connect-api/building-lightning-apps/nwc-js-sdk
- https://guides.getalby.com/developer-guide/developer-guide/bitcoin-payments-agent-skill/references/nwc-client/nwc-client.md
- https://github.com/getAlby/awesome-nwc

### Explicitly out of scope / deprecated for Hub shop design
- https://guides.getalby.com/developer-guide/developer-guide/alby-wallet-api/reference/api-reference/invoices (Alby Wallet API — deprecated)
- https://guides.getalby.com/developer-guide/developer-guide/alby-wallet-api/reference/api-reference/webhook-endpoints (Alby Wallet API webhooks — deprecated)
