# gotype

SSH typing races with optional Lightning tips, and (planned) Player progression plus a non-custodial sats shop.

## Language

**Player**:
A persisted competitor identity with a display name and secret claim code. Owns Inventory, XP, and Season Pass progress.
_Avoid_: account, user, wallet

**Claim Code**:
A 12-character Crockford base32 secret (`XXXX-XXXX-XXXX`) that reclaims a Player over SSH; stored only as a password hash.
_Avoid_: password, API key, seed

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
A voluntary sats payment to the operator via Phoenixd bolt11 (preferred) or LNURL-pay fallback. Not a shop purchase. Phoenixd Tips are tracked as TipIntents until settled.
_Avoid_: donation (in product copy if Tip is the term)

**Buy**:
A sats purchase that unlocks premium shop goods (Cosmetics, Consumables, or Season premium) after payment settles.
_Avoid_: checkout, mint

**Order**:
A server-tracked Buy intent correlated to a Lightning invoice until paid or expired.
_Avoid_: cart, payment request (for the game object)
