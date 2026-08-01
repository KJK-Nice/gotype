package multi

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/kjkusap/monkeytype-clone/internal/game"
)

func TestRedisHubRoundTrip(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("REDIS_URL", "redis://"+mr.Addr())
	h, err := NewHubFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	cfg := game.Config{Mode: game.ModeTime, Duration: 30 * time.Second, WordCount: 25}
	a := NewPlayerID()
	b := NewPlayerID()

	va, err := h.Create(a, "alice", cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Join(b, "bob", va.Code); err != nil {
		t.Fatal(err)
	}

	h2, err := NewHubFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	v := h2.Snapshot(a, time.Now())
	if v.Code != va.Code || len(v.Players) != 2 {
		t.Fatalf("snapshot=%+v", v)
	}
}
