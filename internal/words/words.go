package words

import "math/rand/v2"

// Generate returns n random words from the given word list.
func Generate(list []string, n int) []string {
	return GenerateWithSeed(list, n, rand.Uint64())
}

// GenerateWithSeed returns n words using a deterministic seed (shared multiplayer prompts).
func GenerateWithSeed(list []string, n int, seed uint64) []string {
	if n <= 0 {
		return nil
	}
	r := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	out := make([]string, n)
	for i := range out {
		out[i] = list[r.IntN(len(list))]
	}
	return out
}
