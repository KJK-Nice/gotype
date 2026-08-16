# gotype

SSH typing races with optional Lightning tips, and (planned) Player progression plus a non-custodial sats shop.

## Language

**Player**:
A persisted competitor identity with a display name. Login is LNURL-auth via a Lightning wallet. Owns Inventory, XP, and Season Pass progress.
_Avoid_: account, user, wallet (for the Player itself)

**Login**:
LNURL-auth (LUD-04). Scan a QR with a Lightning wallet; a new wallet picks a display name, a returning wallet restores the Player. TUI shortcut: `l`.
_Avoid_: claim, sign-in, OAuth

**Claim Code**:
Legacy 12-character Crockford base32 secret (`XXXX-XXXX-XXXX`) formerly used to reclaim a Player over SSH. Not offered in the TUI; identity is Linking Key. Stored only as a password hash when present.
_Avoid_: password, API key, seed

**Linking Key**:
A domain-specific Lightning wallet public key that identifies a Player for Login (LNURL-auth).
_Avoid_: pubkey, seed, password

**Inventory**:
The Player's held Cosmetics, Consumables, and Pass entitlements.
_Avoid_: bag, stash

**Cosmetic**:
An equippable presentation unlock. Equip slots: Theme, Caret, Title, FX. Does not change race rules.
_Avoid_: skin (alone), vanity

**FX**:
A Cosmetic slot for ambient motion/effects (e.g. Make it Rain freefall glyphs). One equipped at a time.
_Avoid_: VFX, overlay

**Matrix**:
A Theme Cosmetic with digital-rain / green-terminal presentation; free Season track tier 10.
_Avoid_: neo theme

**Make it Rain**:
An FX Cosmetic where characters freefall on screen; premium Season track tier 15.
_Avoid_: rain mode (that’s gameplay), matrix rain (prefer Make it Rain for the FX)

**Consumable**:
A single-use race aid granted to Inventory; spent on use. v1 classes: Reveal, Calm, Retry (solo), Heart (Three-Strike only).
_Avoid_: power-up, buff item

**Three-Strike**:
An opt-in casual race mode where the Player starts with 3 HP; each incorrect character commit costs 1 HP; at 0 HP the Player DNFs. Player-facing lobby name: **hardcore**.
_Avoid_: sudden death

**Heart**:
A Consumable that restores 1 HP in Three-Strike, up to a max of 5 HP.
_Avoid_: health potion, life

**Season**:
A fixed-length progression window with a free track and a premium track.
_Avoid_: battle pass (as the period), event

**Season Pass**:
The Player's progress on the current Season's free and premium tracks, fueled by XP.
_Avoid_: battle pass (prefer Season Pass), BPass

**XP**:
Soft progression points earned from finishing solo or multi races, subject to a soft daily cap.
_Avoid_: points, score (for progression)

**Tip**:
A voluntary sats payment to the operator via LNURL-pay / bolt11 (existing). Not a shop purchase.
_Avoid_: donation (in product copy if Tip is the term)

**Buy**:
A sats purchase that unlocks premium shop goods (Cosmetics, Consumables, or Season premium) after payment settles.
_Avoid_: checkout, mint

**Order**:
A server-tracked Buy intent correlated to a Lightning invoice until paid or expired.
_Avoid_: cart, payment request (for the game object)
