// Package llama decodes stretches of speech with a recognition model that llama-server runs,
// hearing through its audio projector: Qwen3-ASR (docs/recognition.md). The model punctuates what
// it hears, but times none of it.
package llama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"

	"tomoru/internal/asr/batch"
	"tomoru/internal/models"
)

// Caps are how Qwen3-ASR is run. It ends sentences well, hearing both sides of a pause: a stretch
// goes on through pauses of up to a second, and the sentences it ends while the stretch goes on are
// committed from its volatile text (asr.Spec's VolatileSentences). A decode reads the stretch again
// whole, 0.3–0.7 s per 10 s of speech on a GPU: once a second, and the stretch made final at the
// next pause after 15 s.
var Caps = batch.Caps{Volatile: .5, Gap: 1, Longest: 15, Untimed: true}

const rate = 16000 // samples a second of a stretch

// ContextLimit is the most runes of context a decode is given: about 150 tokens more to read.
const ContextLimit = 300

// An answer is at most tokensPerSecond tokens for each second of the stretch, and tokensMore: an
// answer that long is no speech, but the model going on by itself, as it does over a crowd or
// music ("Congratulations!" 250 times for 3 s of cheering). Speech was 3.6 tokens a second, and 7
// at the 99th percentile, of 1,400 decodes of an interview in English and Chinese.
const (
	tokensPerSecond = 12
	tokensMore      = 16
)

// Decoder is llama-server, serving a recognition model at its URL.
type Decoder struct {
	url, grammar string
	context      string // the system message, what the model is told to expect
	client       *http.Client
	done         chan struct{}
	stop         sync.Once
}

// New decodes speech in lang with the server at url, given context: text that helps it hear, as
// Qwen3-ASR takes it in its system message (names, a description), or "".
func New(url, lang, context string) *Decoder {
	return &Decoder{url: url, grammar: grammar(lang), context: context, client: &http.Client{}, done: make(chan struct{})}
}

// grammar limits how the model's answer starts, "language Japanese<asr_text>…", to the language
// or None, which is how it says it heard no speech; "" lets it tell the language itself. The
// answer must not be started for it with the language: then it has to write something for music,
// applause or a crowd too, and writes a sentence of its own, or its context.
func grammar(lang string) string {
	if !slices.Contains(models.Qwen3ASRLangs, lang) {
		return ""
	}
	return `root ::= "language " ("` + display.English.Tags().Name(language.Make(lang)) + `" | "None") [^\x00]*`
}

// Decode decodes a stretch: its text in pieces, a character or a word each, untimed.
func (d *Decoder) Decode(ctx context.Context, pcm []byte) ([]batch.Piece, error) {
	text, err := d.transcribe(ctx, pcm)
	if err != nil {
		return nil, err
	}
	return batch.Pieces(text), nil
}

func (d *Decoder) transcribe(ctx context.Context, pcm []byte) (string, error) {
	audio := map[string]any{"type": "input_audio", "input_audio": map[string]string{
		"data": base64.StdEncoding.EncodeToString(batch.WAV(pcm)), "format": "wav"}}
	// The system message is there as the model was trained, empty with no context: without one
	// it took noise for speech more often.
	messages := []map[string]any{{"role": "system", "content": d.context}, {"role": "user", "content": []any{audio}}}
	limit := tokensMore + tokensPerSecond*len(pcm)/(2*rate)
	request := map[string]any{"messages": messages, "temperature": 0, "max_tokens": limit, "cache_prompt": false}
	if d.grammar != "" {
		request["grammar"] = d.grammar
	}
	body, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		select {
		case <-d.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.url+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	r, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llama: %w", err)
	}
	defer r.Body.Close()
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("llama: %w", err)
	}
	if r.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llama: HTTP %d: %.200s", r.StatusCode, b)
	}
	var out struct {
		Choices []struct {
			Message      struct{ Content string }
			FinishReason string `json:"finish_reason"`
		}
	}
	if err := json.Unmarshal(b, &out); err != nil || len(out.Choices) == 0 {
		return "", fmt.Errorf("llama: an answer without text: %.200s", b)
	}
	if out.Choices[0].FinishReason == "length" {
		return "", nil // longer than speech is: not what it heard
	}
	return answer(out.Choices[0].Message.Content), nil
}

// answer is the text of the model's answer, "language Japanese<asr_text>…": "" where it heard no
// speech ("language None").
func answer(s string) string {
	head, text, found := strings.Cut(s, "<asr_text>")
	if !found || strings.TrimSpace(head) == "language None" {
		return ""
	}
	return strings.TrimSpace(text)
}

// Warm has the server at url decode a second of silence, for the first stretch not to wait: the
// first requests on a GPU took 3–5 s (Vulkan building its programs).
func Warm(ctx context.Context, url string) error {
	d := New(url, "", "")
	defer d.Stop()
	_, err := d.transcribe(ctx, make([]byte, 32000))
	return err
}

// Stop ends the decoder: a decode in flight is canceled. The server goes on, for the next session.
func (d *Decoder) Stop() { d.stop.Do(func() { close(d.done) }) }

// Done is closed once the decoder has stopped.
func (d *Decoder) Done() <-chan struct{} { return d.done }

// Err is nil: a server that fails fails a decode, and ends the runner with it.
func (d *Decoder) Err() error { return nil }

var _ batch.Decoder = (*Decoder)(nil)
