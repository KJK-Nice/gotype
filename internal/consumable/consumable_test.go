package consumable

import (
	"testing"

	"github.com/kjkusap/monkeytype-clone/internal/catalog"
	"github.com/kjkusap/monkeytype-clone/internal/persist"
)

func TestTrySpendReveal(t *testing.T) {
	dir := t.TempDir()
	store, err := persist.Open(dir + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddInventory("p1", catalog.SKUReveal, 2); err != nil {
		t.Fatal(err)
	}
	ctx := Context{Solo: true, Claimed: true, UsedClass: map[string]bool{}}
	eff, err := TrySpend(store, "p1", catalog.SKUReveal, ctx)
	if err != nil || eff != EffectReveal {
		t.Fatalf("eff=%v err=%v", eff, err)
	}
	ctx.UsedClass["reveal"] = true
	if store.InventoryQty("p1", catalog.SKUReveal) != 1 {
		t.Fatal("expected qty 1")
	}
	_, err = TrySpend(store, "p1", catalog.SKUReveal, ctx)
	if err != ErrAlreadyUsed {
		t.Fatalf("want already used, got %v", err)
	}
}

func TestTrySpendMatchPoint(t *testing.T) {
	store, _ := persist.Open(t.TempDir() + "/data.json")
	_ = store.AddInventory("p1", catalog.SKUReveal, 1)
	_, err := TrySpend(store, "p1", catalog.SKUReveal, Context{
		Solo: true, Claimed: true, MatchPoint: true, UsedClass: map[string]bool{},
	})
	if err != ErrMatchPoint {
		t.Fatalf("got %v", err)
	}
}

func TestTrySpendRetrySoloOnly(t *testing.T) {
	store, _ := persist.Open(t.TempDir() + "/data.json")
	_ = store.AddInventory("p1", catalog.SKURetry, 1)
	_, err := TrySpend(store, "p1", catalog.SKURetry, Context{
		Claimed: true, Solo: false, UsedClass: map[string]bool{},
	})
	if err != ErrSoloOnly {
		t.Fatalf("got %v", err)
	}
}

func TestTrySpendHeartHardcore(t *testing.T) {
	store, _ := persist.Open(t.TempDir() + "/data.json")
	_ = store.AddInventory("p1", catalog.SKUHeart, 1)
	_, err := TrySpend(store, "p1", catalog.SKUHeart, Context{
		Solo: true, Claimed: true, UsedClass: map[string]bool{},
	})
	if err != ErrHardcoreOnly {
		t.Fatalf("got %v", err)
	}
	_, err = TrySpend(store, "p1", catalog.SKUHeart, Context{
		Solo: true, Claimed: true, ThreeStrike: true, HP: 5, MaxHP: 5, UsedClass: map[string]bool{},
	})
	if err != ErrHeartFull {
		t.Fatalf("got %v", err)
	}
}

func TestSpendInventory(t *testing.T) {
	store, _ := persist.Open(t.TempDir() + "/data.json")
	_ = store.AddInventory("p1", "heart", 1)
	if err := store.SpendInventory("p1", "heart", 1); err != nil {
		t.Fatal(err)
	}
	if store.InventoryQty("p1", "heart") != 0 {
		t.Fatal("qty")
	}
	if err := store.SpendInventory("p1", "heart", 1); err != persist.ErrInsufficientQty {
		t.Fatalf("got %v", err)
	}
}
