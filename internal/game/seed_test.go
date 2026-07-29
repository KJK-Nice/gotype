package game

import "testing"

func TestSeededSessionsMatch(t *testing.T) {
	cfg := DefaultConfig
	seed := uint64(42)
	a := NewSessionSeeded(cfg, seed)
	b := NewSessionSeeded(cfg, seed)
	if len(a.Words) == 0 || len(a.Words) != len(b.Words) {
		t.Fatalf("len a=%d b=%d", len(a.Words), len(b.Words))
	}
	for i := range a.Words {
		if a.Words[i] != b.Words[i] {
			t.Fatalf("word[%d]: %q vs %q", i, a.Words[i], b.Words[i])
		}
	}
	c := NewSessionSeeded(cfg, seed+1)
	if c.Words[0] == a.Words[0] && c.Words[1] == a.Words[1] && c.Words[2] == a.Words[2] {
		// extremely unlikely with different seed; soft check first word only often differs
		same := true
		for i := 0; i < 10 && i < len(a.Words); i++ {
			if c.Words[i] != a.Words[i] {
				same = false
				break
			}
		}
		if same {
			t.Fatal("different seeds produced identical prefix")
		}
	}
}
