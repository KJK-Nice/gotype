package words

import "math/rand/v2"

// Generate returns n random words from the given word list.
func Generate(list []string, n int) []string {
	if n <= 0 {
		return nil
	}
	out := make([]string, n)
	for i := range out {
		out[i] = list[rand.IntN(len(list))]
	}
	return out
}
