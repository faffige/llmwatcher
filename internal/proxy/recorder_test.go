package proxy

import (
	"bytes"
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

	// Wait for the async parse goroutine to submit to the sink.
	select {
	case rec := <-sinkCh:
		if rec.Provider != "openai" {
			t.Errorf("sink provider = %q, want %q", rec.Provider, "openai")
		}
		if rec.InputTokens != 5 {
			t.Errorf("sink input_tokens = %d, want 5", rec.InputTokens)
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
