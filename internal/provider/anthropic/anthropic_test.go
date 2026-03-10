package anthropic_test

import (
	"testing"

	"github.com/faffige/llmwatcher/internal/provider/anthropic"
)

func TestParse_Messages(t *testing.T) {
	p := anthropic.New()

	reqBody := []byte(`{
		"model": "claude-sonnet-4-20250514",
		"max_tokens": 1024,
		"messages": [{"role": "user", "content": "Hello"}]
	}`)

	respBody := []byte(`{
		"id": "msg_01XFDUDYJgAACzvnptvVoYEL",
		"type": "message",
		"role": "assistant",
		"model": "claude-sonnet-4-20250514",
		"content": [{"type": "text", "text": "Hi!"}],
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 10,
			"output_tokens": 20
		}
	}`)

	rec := p.Parse("POST", "/v1/messages", 200, reqBody, respBody)
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "anthropic" {
		t.Errorf("provider = %q, want %q", rec.Provider, "anthropic")
	}
	if rec.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", rec.Model, "claude-sonnet-4-20250514")
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

func TestParse_NonMessagesEndpoint_ReturnsNil(t *testing.T) {
	p := anthropic.New()
	rec := p.Parse("POST", "/v1/complete", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-messages endpoint, got %+v", rec)
	}
}

func TestParse_GETRequest_ReturnsNil(t *testing.T) {
	p := anthropic.New()
	rec := p.Parse("GET", "/v1/messages", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for GET request, got %+v", rec)
	}
}

func TestParse_MalformedResponse(t *testing.T) {
	p := anthropic.New()
	rec := p.Parse("POST", "/v1/messages", 200, nil, []byte(`not json`))
	if rec == nil {
		t.Fatal("expected a record even with malformed response")
	}
	if rec.Model != "" {
		t.Errorf("model should be empty for malformed response, got %q", rec.Model)
	}
}

func TestParse_ErrorResponse(t *testing.T) {
	p := anthropic.New()
	respBody := []byte(`{"type": "error", "error": {"type": "authentication_error", "message": "invalid x-api-key"}}`)
	rec := p.Parse("POST", "/v1/messages", 401, nil, respBody)
	if rec == nil {
		t.Fatal("expected a record for error response")
	}
	if rec.StatusCode != 401 {
		t.Errorf("status_code = %d, want 401", rec.StatusCode)
	}
	if rec.ErrorType != "authentication_error" {
		t.Errorf("error_type = %q, want %q", rec.ErrorType, "authentication_error")
	}
	if rec.ErrorMsg != "invalid x-api-key" {
		t.Errorf("error_msg = %q, want %q", rec.ErrorMsg, "invalid x-api-key")
	}
}

func TestProviderName(t *testing.T) {
	p := anthropic.New()
	if name := p.ProviderName(); name != "anthropic" {
		t.Errorf("ProviderName() = %q, want %q", name, "anthropic")
	}
}
