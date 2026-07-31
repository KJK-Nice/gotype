package catalog

// Slot is an equip slot (one Cosmetic each).
type Slot string

const (
	SlotTheme Slot = "theme"
	SlotCaret Slot = "caret"
	SlotTitle Slot = "title"
	SlotFX    Slot = "fx"
)

// Kind classifies a catalog SKU.
type Kind string

const (
	KindCosmetic   Kind = "cosmetic"
	KindConsumable Kind = "consumable"
	KindPremium    Kind = "season_premium"
)

// SKU identifiers (config/code, not DB rows).
const (
	SKUMatrix       = "matrix"
	SKUMakeItRain   = "make_it_rain"
	SKUReveal       = "reveal"
	SKUCalm         = "calm"
	SKURetry        = "retry"
	SKUHeart        = "heart"
	SKUSeasonPrem   = "season_premium"
)

// Item is a catalog entry.
type Item struct {
	SKU   string
	Name  string
	Kind  Kind
	Slot  Slot // cosmetics only
	Sats  int  // shop price; 0 if not sold
	Class string // consumable class
}

// ShopItems are Buyable SKUs (Consumables + Season premium).
func ShopItems() []Item {
	return []Item{
		{SKU: SKUReveal, Name: "Reveal", Kind: KindConsumable, Sats: 21, Class: "reveal"},
		{SKU: SKUCalm, Name: "Calm", Kind: KindConsumable, Sats: 21, Class: "calm"},
		{SKU: SKURetry, Name: "Retry", Kind: KindConsumable, Sats: 50, Class: "retry"},
		{SKU: SKUHeart, Name: "Heart", Kind: KindConsumable, Sats: 100, Class: "heart"},
		{SKU: SKUSeasonPrem, Name: "Season premium", Kind: KindPremium, Sats: 2100},
	}
}

// NamedCosmetics are Season-track Cosmetics (not shop-sold).
func NamedCosmetics() []Item {
	return []Item{
		{SKU: SKUMatrix, Name: "Matrix", Kind: KindCosmetic, Slot: SlotTheme},
		{SKU: SKUMakeItRain, Name: "Make it Rain", Kind: KindCosmetic, Slot: SlotFX},
	}
}

// Lookup returns a catalog item by SKU.
func Lookup(sku string) (Item, bool) {
	for _, it := range ShopItems() {
		if it.SKU == sku {
			return it, true
		}
	}
	for _, it := range NamedCosmetics() {
		if it.SKU == sku {
			return it, true
		}
	}
	return Item{}, false
}

// SeasonLengthDays is the progression window length.
const SeasonLengthDays = 60

// FreeTrackReward maps free-track tier → Cosmetic SKU (v1 named only).
func FreeTrackReward(tier int) string {
	if tier == 10 {
		return SKUMatrix
	}
	return ""
}

// PremiumTrackReward maps premium-track tier → Cosmetic SKU.
func PremiumTrackReward(tier int) string {
	if tier == 15 {
		return SKUMakeItRain
	}
	return ""
}
