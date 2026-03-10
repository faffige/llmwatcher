package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/faffige/llmwatcher/internal/provider"
	"github.com/faffige/llmwatcher/internal/provider/openai"
)

func TestRecorder_CapturesAndProxies(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := openai.New()

	sinkCh := make(chan *provider.CallRecord, 1)
	sink := func(rec *provider.CallRecord) {
		sinkCh <- rec
	}

	handler := newRecorder(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("gpt-4o")) {
			t.Errorf("inner handler did not receive expected request body, got: %s", body)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"model": "gpt-4o-2024-08-06",
			"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
		}`))
	}), parser, sink, logger)

	reqBody := `{"model": "gpt-4o", "messages": [{"role": "user", "content": "hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	if !bytes.Contains(rr.Body.Bytes(), []byte("gpt-4o-2024-08-06")) {
		t.Errorf("response body not passed through, got: %s", rr.Body.String())
	}

	select {
	case rec := <-sinkCh:
		if rec.Provider != "openai" {
			t.Errorf("sink provider = %q, want %q", rec.Provider, "openai")
		}
		if rec.InputTokens != 5 {
			t.Errorf("sink input_tokens = %d, want 5", rec.InputTokens)
		}
		if rec.IsStream {
			t.Error("expected IsStream = false for non-streaming request")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for record on sink")
	}
}

func TestRecorder_StreamingRequest(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	parser := openai.New()

	sinkCh := make(chan *provider.CallRecord, 1)
	sink := func(rec *provider.CallRecord) {
		sinkCh <- rec
	}

	sseResp := "data: {\"model\":\"gpt-4o-2024-08-06\",\"choices\":[{\"delta\":{\"content\":\"Hi\"}}]}\n\n" +
		"data: {\"model\":\"gpt-4o-2024-08-06\",\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":1,\"total_tokens\":4}}\n\n" +
		"data: [DONE]\n\n"

	handler := newRecorder(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify stream_options.include_usage was injected.
		body, _ := io.ReadAll(r.Body)
		var req map[string]json.RawMessage
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("failed to parse forwarded request: %v", err)
			return
		}
		var opts map[string]any
		if raw, ok := req["stream_options"]; ok {
			json.Unmarshal(raw, &opts)
		}
		if v, ok := opts["include_usage"]; !ok || v != true {
			t.Errorf("stream_options.include_usage not injected, got: %v", opts)
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sseResp))
	}), parser, sink, logger)

	reqBody := `{"model": "gpt-4o", "stream": true, "messages": [{"role": "user", "content": "hi"}]}`
	req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(reqBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	select {
	case rec := <-sinkCh:
		if !rec.IsStream {
			t.Error("expected IsStream = true")
		}
		if rec.InputTokens != 3 {
			t.Errorf("input_tokens = %d, want 3", rec.InputTokens)
		}
		if rec.OutputTokens != 1 {
			t.Errorf("output_tokens = %d, want 1", rec.OutputTokens)
		}
		if rec.Model != "gpt-4o-2024-08-06" {
			t.Errorf("model = %q, want %q", rec.Model, "gpt-4o-2024-08-06")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for record on sink")
	}
}

func TestRecorder_NilParser_PassesThrough(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := newRecorder(inner, nil, nil, logger)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("inner handler was not called")
	}
}

func TestRequestModifier_InjectsIncludeUsage(t *testing.T) {
	parser := openai.New()
	body := []byte(`{"model":"gpt-4o","stream":true}`)
	out := parser.ModifyStreamingRequest(body)

	var req map[string]json.RawMessage
	json.Unmarshal(out, &req)
	var opts map[string]any
	json.Unmarshal(req["stream_options"], &opts)

	if v, ok := opts["include_usage"]; !ok || v != true {
		t.Errorf("include_usage not set, got: %v", opts)
	}
}

func TestRequestModifier_AlreadySet(t *testing.T) {
	parser := openai.New()
	body := []byte(`{"model":"gpt-4o","stream":true,"stream_options":{"include_usage":true}}`)
	out := parser.ModifyStreamingRequest(body)

	// Should be unchanged.
	if !bytes.Equal(body, out) {
		t.Errorf("body was modified when include_usage already set")
	}
}

func TestIsStreamingRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"streaming", `{"stream":true}`, true},
		{"not streaming", `{"stream":false}`, false},
		{"no stream field", `{"model":"gpt-4o"}`, false},
		{"empty", ``, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isStreamingRequest([]byte(tt.body))
			if got != tt.want {
				t.Errorf("isStreamingRequest(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}
