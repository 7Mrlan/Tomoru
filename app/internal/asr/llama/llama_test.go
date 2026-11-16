package llama

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAnswer(t *testing.T) {
	for in, want := range map[string]string{
		"language Japanese<asr_text>その後、大事はないか。": "その後、大事はないか。",
		"language None<asr_text>":                "",
		"language None":                          "",
		" language English<asr_text> Hello.":     "Hello.",
	} {
		if got := answer(in); got != want {
			t.Errorf("answer(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestGrammar leaves the model its language or None, no speech, to start its answer with.
func TestGrammar(t *testing.T) {
	for lang, want := range map[string]string{"ja": `root ::= "language " ("Japanese" | "None") [^\x00]*`,
		"yue": `root ::= "language " ("Cantonese" | "None") [^\x00]*`, "zh-Hant": "", "xx": ""} {
		if got := grammar(lang); got != want {
			t.Errorf("grammar(%q) = %q, want %q", lang, got, want)
		}
	}
}

// TestDecode sends the stretch as a WAV, the model to tell the language with its likeliest
// alternatives, and reads the answer.
func TestDecode(t *testing.T) {
	var got struct {
		Messages []struct {
			Role    string
			Content json.RawMessage
		}
		Grammar     string
		Logprobs    bool
		TopLogprobs int `json:"top_logprobs"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"choices":[{"message":{"content":"language Japanese<asr_text>狼よ。"}}]}`))
	}))
	defer srv.Close()
	d := New(srv.URL, "ja", "一心, 葦名")
	defer d.Stop()
	ps, err := d.Decode(context.Background(), make([]byte, 3200))
	if err != nil || len(ps) != 3 || ps[0].Text != "狼" || ps[2].Text != "。" {
		t.Fatalf("%v %v", ps, err)
	}
	var system string
	if len(got.Messages) == 2 {
		json.Unmarshal(got.Messages[0].Content, &system)
	}
	if len(got.Messages) != 2 || got.Messages[0].Role != "system" || system != "一心, 葦名" || got.Messages[1].Role != "user" || got.Grammar != "" || !got.Logprobs || got.TopLogprobs != topLogprobs {
		t.Fatalf("messages %d, system %q, grammar %q, logprobs %v %d", len(got.Messages), system, got.Grammar, got.Logprobs, got.TopLogprobs)
	}
	got.Messages = got.Messages[1:] // the user's, with the audio, first
	var parts []struct {
		Type       string
		InputAudio struct{ Data, Format string } `json:"input_audio"`
	}
	json.Unmarshal(got.Messages[0].Content, &parts)
	audio, _ := base64.StdEncoding.DecodeString(parts[0].InputAudio.Data)
	if len(parts) != 1 || parts[0].Type != "input_audio" || parts[0].InputAudio.Format != "wav" ||
		string(audio[:4]) != "RIFF" || string(audio[8:12]) != "WAVE" || len(audio) != 44+3200 {
		t.Fatalf("audio %+v, %d bytes", parts, len(audio))
	}
}

// TestLanguage keeps another language than the session's when the model is sure of it, and decodes
// again in the session's language when not.
func TestLanguage(t *testing.T) {
	const other = `{"choices":[{"message":{"content":"language Chinese<asr_text>我们BLG就是最强的。"},"logprobs":{"content":[` +
		`{"token":"language","top_logprobs":[{"token":"language","logprob":0}]},` +
		`{"token":" Chinese","top_logprobs":[{"token":" Chinese","logprob":0},{"token":" Cantonese","logprob":-9}%s]}]}}]}`
	for _, c := range []struct {
		name, alternatives string
		asks               int
		want               string
	}{
		{"sure", `,{"token":" Eng","logprob":-20}`, 1, "我们BLG就是最强的。"},
		{"not sure", `,{"token":" English","logprob":-4}`, 2, "We BLG are the strongest."},
		{"not listed, below the least listed", "", 2, "We BLG are the strongest."},
		{"not listed, the least listed unlikely", `,{"token":" Japanese","logprob":-20}`, 1, "我们BLG就是最强的。"},
	} {
		asks := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var got struct{ Grammar string }
			json.NewDecoder(r.Body).Decode(&got)
			asks++
			if got.Grammar == grammar("en") {
				w.Write([]byte(`{"choices":[{"message":{"content":"language English<asr_text>We BLG are the strongest."}}]}`))
				return
			}
			fmt.Fprintf(w, other, c.alternatives)
		}))
		d := New(srv.URL, "en", "")
		text, err := d.transcribe(context.Background(), make([]byte, 3200))
		d.Stop()
		srv.Close()
		if err != nil || text != c.want || asks != c.asks {
			t.Errorf("%s: %q after %d asks, %v", c.name, text, asks, err)
		}
	}
}

// TestBound gives an answer as many tokens as the stretch's speech may take, and takes one that
// used them all as no speech: the model going on by itself. An empty context is still sent, as the
// model was trained.
func TestBound(t *testing.T) {
	var got struct {
		Messages  []struct{ Role string }
		MaxTokens int `json:"max_tokens"`
	}
	reason := "length"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&got)
		w.Write([]byte(`{"choices":[{"finish_reason":"` + reason + `","message":{"content":"language English<asr_text>Congratulations! Congratulations!"}}]}`))
	}))
	defer srv.Close()
	d := New(srv.URL, "en", "")
	defer d.Stop()
	ps, err := d.Decode(context.Background(), make([]byte, 3*2*rate))
	if err != nil || len(ps) != 0 {
		t.Fatalf("an answer cut at its limit: %v %v", ps, err)
	}
	if got.MaxTokens != tokensMore+3*tokensPerSecond || len(got.Messages) != 2 || got.Messages[0].Role != "system" {
		t.Fatalf("max_tokens %d for 3 s, messages %+v", got.MaxTokens, got.Messages)
	}
	reason = "stop"
	if ps, err := d.Decode(context.Background(), make([]byte, 3*2*rate)); err != nil || len(ps) == 0 {
		t.Fatalf("an answer that ended: %v %v", ps, err)
	}
}

// TestStop cancels a decode in flight, and a server error fails a decode.
func TestStop(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	d := New(srv.URL, "en", "")
	errc := make(chan error, 1)
	go func() { _, err := d.Decode(context.Background(), nil); errc <- err }()
	time.Sleep(50 * time.Millisecond)
	d.Stop()
	select {
	case err := <-errc:
		if err == nil {
			t.Fatal("a stopped decode succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not cancel the decode")
	}
	<-d.Done()

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "loading model", http.StatusServiceUnavailable)
	}))
	defer failing.Close()
	if _, err := New(failing.URL, "en", "").Decode(context.Background(), nil); err == nil || errors.Is(err, context.Canceled) {
		t.Fatalf("a server error: %v", err)
	}
}
