package roast

import (
	"strings"
	"testing"
	"time"
)

func TestLocalZeroKeys(t *testing.T) {
	out := Local(Input{TypedAny: false})
	if out == "" {
		t.Fatal("expected roast")
	}
}

func TestLocalBands(t *testing.T) {
	cases := []Input{
		{TypedAny: true, WPM: 20, Accuracy: 50, Wrong: 40, Detail: "15s", Mode: "time", Elapsed: 15 * time.Second},
		{TypedAny: true, WPM: 60, Accuracy: 95, Wrong: 2, Detail: "30s", Mode: "time", Elapsed: 30 * time.Second},
		{TypedAny: true, WPM: 140, Accuracy: 98, Wrong: 1, Detail: "25 words", Mode: "words", Elapsed: 20 * time.Second},
	}
	for _, in := range cases {
		out := Local(in)
		if strings.TrimSpace(out) == "" {
			t.Fatalf("empty roast for %+v", in)
		}
	}
}

func TestLocalStoic(t *testing.T) {
	out := Local(Input{TypedAny: true, WPM: 70, Accuracy: 96, Voice: VoiceStoic, Detail: "30s"})
	if strings.TrimSpace(out) == "" {
		t.Fatal("expected stoic line")
	}
}

func TestProviderGeminiKey(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ROAST_API_KEY", "")
	t.Setenv("ROAST_PROVIDER", "")
	t.Setenv("GOOGLE_API_KEY", "")
	if !Configured() {
		t.Fatal("expected configured")
	}
	if ProviderName() != "google" {
		t.Fatalf("got %q want google", ProviderName())
	}
}

func TestProviderOpenAIExplicit(t *testing.T) {
	t.Setenv("ROAST_PROVIDER", "openai")
	t.Setenv("ROAST_API_KEY", "sk-test")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GOOGLE_API_KEY", "")
	if ProviderName() != "openai" {
		t.Fatalf("got %q want openai", ProviderName())
	}
}
