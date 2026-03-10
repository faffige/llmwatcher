package openai_test

import (
	"testing"

	"github.com/faffige/llmwatcher/internal/provider/openai"
)

func TestParse_ChatCompletion(t *testing.T) {
	p := openai.New()

	reqBody := []byte(`{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "Hello"}]
	}`)

	respBody := []byte(`{
		"id": "chatcmpl-abc123",
		"object": "chat.completion",
		"model": "gpt-4o-2024-08-06",
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 20,
			"total_tokens": 30
		},
		"choices": [
			{"index": 0, "message": {"role": "assistant", "content": "Hi!"}}
		]
	}`)

	rec := p.Parse("POST", "/v1/chat/completions", 200, reqBody, respBody)
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "openai" {
		t.Errorf("provider = %q, want %q", rec.Provider, "openai")
	}
	if rec.Model != "gpt-4o-2024-08-06" {
		t.Errorf("model = %q, want %q", rec.Model, "gpt-4o-2024-08-06")
	}
	if rec.Operation != "chat" {
		t.Errorf("operation = %q, want %q", rec.Operation, "chat")
	}
	if rec.InputTokens != 10 {
		t.Errorf("input_tokens = %d, want 10", rec.InputTokens)
	}
	if rec.OutputTokens != 20 {
		t.Errorf("output_tokens = %d, want 20", rec.OutputTokens)
	}
	if rec.TotalTokens != 30 {
		t.Errorf("total_tokens = %d, want 30", rec.TotalTokens)
	}
	if rec.StatusCode != 200 {
		t.Errorf("status_code = %d, want 200", rec.StatusCode)
	}
}

func TestParse_NonChatEndpoint_ReturnsNil(t *testing.T) {
	p := openai.New()
	rec := p.Parse("POST", "/v1/embeddings", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-chat endpoint, got %+v", rec)
	}
}

func TestParse_GETRequest_ReturnsNil(t *testing.T) {
	p := openai.New()
	rec := p.Parse("GET", "/v1/chat/completions", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for GET request, got %+v", rec)
	}
}

func TestParse_MalformedResponse(t *testing.T) {
	p := openai.New()
	rec := p.Parse("POST", "/v1/chat/completions", 200, nil, []byte(`not json`))
	if rec == nil {
		t.Fatal("expected a record even with malformed response")
	}
	if rec.Model != "" {
		t.Errorf("model should be empty for malformed response, got %q", rec.Model)
	}
}

func TestParse_ErrorResponse(t *testing.T) {
	p := openai.New()
	respBody := []byte(`{"error": {"message": "invalid api key", "type": "invalid_request_error"}}`)
	rec := p.Parse("POST", "/v1/chat/completions", 401, nil, respBody)
	if rec == nil {
		t.Fatal("expected a record for error response")
	}
	if rec.StatusCode != 401 {
		t.Errorf("status_code = %d, want 401", rec.StatusCode)
	}
	if rec.ErrorType != "invalid_request_error" {
		t.Errorf("error_type = %q, want %q", rec.ErrorType, "invalid_request_error")
	}
	if rec.ErrorMsg != "invalid api key" {
		t.Errorf("error_msg = %q, want %q", rec.ErrorMsg, "invalid api key")
	}
}

func TestProviderName(t *testing.T) {
	p := openai.New()
	if name := p.ProviderName(); name != "openai" {
		t.Errorf("ProviderName() = %q, want %q", name, "openai")
	}
}
