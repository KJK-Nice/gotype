package roast

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

// Voice is the result commentary style.
type Voice int

const (
	VoiceRoast Voice = iota
	VoiceStoic
)

func (v Voice) String() string {
	switch v {
	case VoiceStoic:
		return "stoic"
	default:
		return "roast"
	}
}

// Input is enough run stats for a roast/stoic line.
type Input struct {
	WPM      float64
	RawWPM   float64
	Accuracy float64
	Correct  int
	Wrong    int
	Elapsed  time.Duration
	Mode     string // "time" or "words"
	Detail   string // e.g. "30s" or "25 words"
	TypedAny bool   // false if zero keystrokes
	Voice    Voice
}

type provider string

const (
	providerOpenAI provider = "openai"
	providerGoogle provider = "google"

	defaultOpenAIBase = "https://api.openai.com/v1"
	defaultOpenAIModel = "gpt-4o-mini"
	defaultGeminiModel = "gemini-2.5-flash-lite"
	geminiBaseURL      = "https://generativelanguage.googleapis.com/v1beta"
)

const (
	systemRoast = "You roast typing-test results for a terminal typing game called gotype. " +
		"Be witty, mean-but-playful, one or two short sentences max. No markdown, no quotes, no hashtags, no emoji spam. " +
		"If they crushed it, roast the ego anyway. If they bombed, be creative not cruel about identity."

	systemStoic = "You speak as a calm Stoic mentor commenting on a typing-test result in a terminal app called gotype. " +
		"Tone: Marcus Aurelius / Epictetus — brief, grounded, unsentimental. One or two short sentences max. " +
		"No markdown, no quotes wrapping the whole reply, no hashtags, no emoji. " +
		"Praise without flattery; critique without cruelty. Focus on effort, attention, and what is in their control."
)

func systemPrompt(v Voice) string {
	if v == VoiceStoic {
		return systemStoic
	}
	return systemRoast
}

// Configured reports whether an LLM API key is set.
func Configured() bool {
	return apiKey() != ""
}

// ProviderName returns active backend: "google", "openai", or "".
func ProviderName() string {
	if !Configured() {
		return ""
	}
	return string(activeProvider())
}

func activeProvider() provider {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("ROAST_PROVIDER"))) {
	case "google", "gemini":
		return providerGoogle
	case "openai":
		return providerOpenAI
	}
	if os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("GOOGLE_API_KEY") != "" {
		return providerGoogle
	}
	return providerOpenAI
}

func apiKey() string {
	if k := os.Getenv("ROAST_API_KEY"); k != "" {
		return k
	}
	if activeProvider() == providerGoogle {
		if k := os.Getenv("GEMINI_API_KEY"); k != "" {
			return k
		}
		if k := os.Getenv("GOOGLE_API_KEY"); k != "" {
			return k
		}
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		return k
	}
	// Allow Gemini keys even if provider not forced yet.
	if k := os.Getenv("GEMINI_API_KEY"); k != "" {
		return k
	}
	return os.Getenv("GOOGLE_API_KEY")
}

func baseURL() string {
	if u := os.Getenv("ROAST_BASE_URL"); u != "" {
		return strings.TrimRight(u, "/")
	}
	return defaultOpenAIBase
}

func model() string {
	if m := os.Getenv("ROAST_MODEL"); m != "" {
		return m
	}
	if activeProvider() == providerGoogle {
		return defaultGeminiModel
	}
	return defaultOpenAIModel
}

// Generate returns a short roast. Uses LLM when configured; else local templates.
func Generate(ctx context.Context, in Input) (string, error) {
	if !Configured() {
		return Local(in), nil
	}
	text, err := callLLM(ctx, in)
	if err != nil {
		return Local(in), nil // soft-fail to canned roast
	}
	return text, nil
}

// Local picks a canned line from stats (no network).
func Local(in Input) string {
	if in.Voice == VoiceStoic {
		return localStoic(in)
	}
	return localRoast(in)
}

func localRoast(in Input) string {
	if !in.TypedAny {
		return pick([]string{
			"zero keys. bold strategy. the keyboard files a missing-person report.",
			"you opened a typing test and meditated. monk mode unlocked; wpm remains rumor.",
			"afk speedrun complete. the words are still waiting. so is dignity.",
		})
	}

	acc := in.Accuracy
	wpm := in.WPM
	wrong := in.Wrong

	switch {
	case acc < 70:
		return pick([]string{
			fmt.Sprintf("%.0f%% accuracy. that was less typing, more interpretive dance on the keys.", acc),
			fmt.Sprintf("%d wrongs. autocorrect filed for divorce. you still hit enter like a hero.", wrong),
			fmt.Sprintf("%.0f wpm of pure chaos. speed is optional when accuracy is folklore.", wpm),
		})
	case acc < 90:
		return pick([]string{
			fmt.Sprintf("%.0f wpm @ %.0f%%. decent, if \"decent\" means your fingers argue mid-sentence.", wpm, acc),
			fmt.Sprintf("%d mistakes sneaking in. the chart looks like a heart monitor after espresso.", wrong),
			fmt.Sprintf("%.0f wpm — almost proud, then the error dots started gossiping.", wpm),
		})
	case wpm < 40:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. glacial. tectonic plates type hate mail faster.", wpm),
			fmt.Sprintf("%.0f%% accurate and still %.0f wpm. correct… slowly… like a notary.", acc, wpm),
			fmt.Sprintf("pace ghost took a nap waiting for you. %.0f wpm lullaby.", wpm),
		})
	case wpm < 80:
		return pick([]string{
			fmt.Sprintf("%.0f wpm / %.0f%%. solid mid. not viral. not tragic. aggressively fine.", wpm, acc),
			fmt.Sprintf("%.0f wpm — the typing equivalent of a firm handshake and mild eye contact.", wpm),
			fmt.Sprintf("%s · %.0f wpm. monkeytype would nod once, then forget your name.", in.Detail, wpm),
		})
	case wpm < 120:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. spicy. your coworkers' slack is in danger.", wpm),
			fmt.Sprintf("%.0f wpm @ %.0f%%. keys filed a noise complaint. neighbors learned english.", wpm, acc),
			fmt.Sprintf("hot streak energy. %.0f wpm — leave some alphabet for the rest of us.", wpm),
		})
	default:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. inhuman. captcha just surrendered.", wpm),
			fmt.Sprintf("%.0f wpm @ %.0f%%. the server briefly considered rate-limiting your fingers.", wpm, acc),
			fmt.Sprintf("elite. %.0f wpm. go touch grass — carefully, at 200wpm.", wpm),
		})
	}
}

func localStoic(in Input) string {
	if !in.TypedAny {
		return pick([]string{
			"silence is also a choice. the keys waited; so can you. begin when ready.",
			"no stroke was made. the obstacle was not the test — only the decision to start.",
			"rest is not failure. return, and let the next letter be enough.",
		})
	}

	acc := in.Accuracy
	wpm := in.WPM
	wrong := in.Wrong

	switch {
	case acc < 70:
		return pick([]string{
			fmt.Sprintf("%.0f%% true. haste without aim multiplies error. slow the hand; keep the mind.", acc),
			fmt.Sprintf("%d missteps. each one is instruction, if you refuse to look away.", wrong),
			fmt.Sprintf("%.0f wpm of unrest. speed outside virtue is only noise.", wpm),
		})
	case acc < 90:
		return pick([]string{
			fmt.Sprintf("%.0f wpm, %.0f%%. progress is uneven — that is the nature of practice.", wpm, acc),
			fmt.Sprintf("%d faults remain. attend to what you control: the next correct key.", wrong),
			fmt.Sprintf("%.0f wpm with room to refine. do not despise the unfinished self.", wpm),
		})
	case wpm < 40:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. patience is not delay — it is mastery refused to rush.", wpm),
			fmt.Sprintf("%.0f%% held at %.0f wpm. accuracy first; pace follows the trained will.", acc, wpm),
			fmt.Sprintf("the ghost waits without judgment. %.0f wpm is still motion toward the good.", wpm),
		})
	case wpm < 80:
		return pick([]string{
			fmt.Sprintf("%.0f wpm / %.0f%%. steady work. neither glory nor shame — only the task.", wpm, acc),
			fmt.Sprintf("%.0f wpm. enough. return tomorrow as if the score were ash.", wpm),
			fmt.Sprintf("%s done. measure yourself against yesterday's effort, not the crowd.", in.Detail),
		})
	case wpm < 120:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. strength shown. do not let it become vanity at the next prompt.", wpm),
			fmt.Sprintf("%.0f wpm @ %.0f%%. skill is a tool; character is how you hold it.", wpm, acc),
			"swift hands. keep the mind quieter still than the keys.",
		})
	default:
		return pick([]string{
			fmt.Sprintf("%.0f wpm. excellence without attachment. the next run starts at zero.", wpm),
			fmt.Sprintf("%.0f wpm @ %.0f%%. fortune favored preparation. remain indifferent to praise.", wpm, acc),
			fmt.Sprintf("%.0f wpm. what is high can fall. what is practiced endures.", wpm),
		})
	}
}

func pick(opts []string) string {
	if len(opts) == 0 {
		return "that run happened. emotionally, we decline comment."
	}
	return opts[rand.Intn(len(opts))]
}

func userPrompt(in Input) string {
	return fmt.Sprintf(
		"typing test result:\nmode: %s (%s)\nwpm: %.0f\nraw: %.0f\naccuracy: %.0f%%\ncorrect chars: %d\nwrong chars: %d\ntime: %.1fs\ntyped: %v",
		in.Mode, in.Detail, in.WPM, in.RawWPM, in.Accuracy, in.Correct, in.Wrong, in.Elapsed.Seconds(), in.TypedAny,
	)
}

func callLLM(ctx context.Context, in Input) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	switch activeProvider() {
	case providerGoogle:
		return callGemini(ctx, in)
	default:
		return callOpenAI(ctx, in)
	}
}

func callOpenAI(ctx context.Context, in Input) (string, error) {
	body := map[string]any{
		"model": model(),
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt(in.Voice)},
			{"role": "user", "content": userPrompt(in)},
		},
		"temperature": 0.95,
		"max_tokens":  80,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL()+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey())

	data, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("empty roast response")
	}
	return cleanRoast(parsed.Choices[0].Message.Content)
}

func callGemini(ctx context.Context, in Input) (string, error) {
	body := map[string]any{
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": systemPrompt(in.Voice)}},
		},
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": userPrompt(in)}},
			},
		},
		"generationConfig": map[string]any{
			"temperature":     0.95,
			"maxOutputTokens": 80,
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/models/%s:generateContent", geminiBaseURL, model())
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey())

	data, err := doHTTP(req)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("gemini: %s", parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty gemini roast")
	}
	var b strings.Builder
	for _, p := range parsed.Candidates[0].Content.Parts {
		b.WriteString(p.Text)
	}
	return cleanRoast(b.String())
}

func doHTTP(req *http.Request) ([]byte, error) {
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("roast api %s: %s", res.Status, truncate(string(data), 180))
	}
	return data, nil
}

func cleanRoast(text string) (string, error) {
	text = strings.TrimSpace(text)
	text = strings.Trim(text, "\"'`")
	if text == "" {
		return "", fmt.Errorf("blank roast")
	}
	return text, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
