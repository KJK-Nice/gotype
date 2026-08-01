package persist

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func openTestRedis(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	s, err := OpenRedis("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	return s, mr
}

func TestRedisStoreRoundTrip(t *testing.T) {
	s, mr := openTestRedis(t)
	runStoreRoundTrip(t, s)

	s2, err := OpenRedis("redis://" + mr.Addr())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.GetPlayer("p1")
	if err != nil || got.Name != "Neo" {
		t.Fatalf("%+v %v", got, err)
	}
	if s2.InventoryQty("p1", "heart") != 2 {
		t.Fatal("inventory not persisted in redis")
	}
}

func TestRedisGrantPaidOrderIdempotent(t *testing.T) {
	s, _ := openTestRedis(t)
	runGrantPaidOrderIdempotent(t, s)
}

func TestRedisApplyRewardClaimsIdempotent(t *testing.T) {
	s, _ := openTestRedis(t)
	runApplyRewardClaimsIdempotent(t, s)
}
