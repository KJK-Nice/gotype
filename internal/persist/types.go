package persist

import "time"

// Player is a persisted competitor identity.
type Player struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	NameKey         string    `json:"name_key"`
	ClaimHash       string    `json:"claim_hash"`
	LinkingKey      string    `json:"linking_key,omitempty"`
	RenameUsed      bool      `json:"rename_used"`
	ActiveSessionID string    `json:"active_session_id"`
	CreatedAt       time.Time `json:"created_at"`
	BestCombo       int       `json:"best_combo,omitempty"`
}

// InventoryItem is Player + SKU + qty/owned.
type InventoryItem struct {
	PlayerID string `json:"player_id"`
	SKU      string `json:"sku"`
	Qty      int    `json:"qty"` // cosmetics: 1 owned; consumables: stack
}

// Equipment maps Player + slot → SKU.
type Equipment struct {
	PlayerID string `json:"player_id"`
	Slot     string `json:"slot"`
	SKU      string `json:"sku"`
}

// Season is a progression window.
type Season struct {
	ID       int       `json:"id"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt   time.Time `json:"ends_at"`
	TrackRef string    `json:"track_ref"`
}

// SeasonProgress is XP + premium + claimed tiers for one Player/Season.
type SeasonProgress struct {
	PlayerID        string `json:"player_id"`
	SeasonID        int    `json:"season_id"`
	XP              int    `json:"xp"`
	PremiumUnlocked bool   `json:"premium_unlocked"`
	ClaimedFree     []int  `json:"claimed_free"`
	ClaimedPremium  []int  `json:"claimed_premium"`
}

// OrderState is the Buy state machine.
type OrderState string

const (
	OrderCreated  OrderState = "created"
	OrderInvoiced OrderState = "invoiced"
	OrderPaid     OrderState = "paid"
	OrderGranted  OrderState = "granted"
	OrderExpired  OrderState = "expired"
	OrderFailed   OrderState = "failed"
)

// Order is a server-tracked Buy intent.
type Order struct {
	ID          string     `json:"id"`
	PlayerID    string     `json:"player_id"`
	SKU         string     `json:"sku"`
	Sats        int        `json:"sats"`
	State       OrderState `json:"state"`
	Bolt11      string     `json:"bolt11,omitempty"`
	PaymentHash string     `json:"payment_hash,omitempty"`
	CheckingID  string     `json:"checking_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	GrantedAt   time.Time  `json:"granted_at,omitempty"`
}

// DailyXP tracks UTC-day XP toward the soft cap.
type DailyXP struct {
	PlayerID string `json:"player_id"`
	Day      string `json:"day"` // YYYY-MM-DD UTC
	XP       int    `json:"xp"`
}
