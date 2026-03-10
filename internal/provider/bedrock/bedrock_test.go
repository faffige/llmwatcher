package bedrock_test

import (
	"testing"

	"github.com/faffige/llmwatcher/internal/provider/bedrock"
)

func TestParse_InvokeModel(t *testing.T) {
	p := bedrock.New()

	reqBody := []byte(`{
		"anthropic_version": "bedrock-2023-05-31",
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

	rec := p.Parse("POST", "/model/anthropic.claude-sonnet-4-20250514-v1:0/invoke", 200, reqBody, respBody)
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "bedrock" {
		t.Errorf("provider = %q, want %q", rec.Provider, "bedrock")
	}
	if rec.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", rec.Model, "claude-sonnet-4-20250514")
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
}

func TestParse_NonInvokeEndpoint_ReturnsNil(t *testing.T) {
	p := bedrock.New()
	rec := p.Parse("POST", "/model/some-model/converse", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-invoke endpoint, got %+v", rec)
	}
}

func TestParse_GETRequest_ReturnsNil(t *testing.T) {
	p := bedrock.New()
	rec := p.Parse("GET", "/model/some-model/invoke", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for GET request, got %+v", rec)
	}
}

func TestParse_ErrorResponse(t *testing.T) {
	p := bedrock.New()
	respBody := []byte(`{"type": "error", "error": {"type": "authentication_error", "message": "invalid credentials"}}`)
	rec := p.Parse("POST", "/model/anthropic.claude-sonnet-4-20250514-v1:0/invoke", 403, nil, respBody)
	if rec == nil {
		t.Fatal("expected a record for error response")
	}
	if rec.ErrorType != "authentication_error" {
		t.Errorf("error_type = %q, want %q", rec.ErrorType, "authentication_error")
	}
}

func TestParseStream_InvokeWithResponseStream(t *testing.T) {
	p := bedrock.New()

	sse := `event: message_start
data: {"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":8,"output_tokens":0}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}

event: message_stop
data: {"type":"message_stop"}
`

	rec := p.ParseStream("POST", "/model/anthropic.claude-sonnet-4-20250514-v1:0/invoke-with-response-stream", 200, nil, []byte(sse))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "bedrock" {
		t.Errorf("provider = %q, want %q", rec.Provider, "bedrock")
	}
	if !rec.IsStream {
		t.Error("expected IsStream = true")
	}
	if rec.InputTokens != 8 {
		t.Errorf("input_tokens = %d, want 8", rec.InputTokens)
	}
	if rec.OutputTokens != 3 {
		t.Errorf("output_tokens = %d, want 3", rec.OutputTokens)
	}
	if rec.TotalTokens != 11 {
		t.Errorf("total_tokens = %d, want 11", rec.TotalTokens)
	}
}

func TestParseStream_NonStreamEndpoint_ReturnsNil(t *testing.T) {
	p := bedrock.New()
	rec := p.ParseStream("POST", "/model/some-model/invoke", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-stream endpoint, got %+v", rec)
	}
}

func TestProviderName(t *testing.T) {
	p := bedrock.New()
	if name := p.ProviderName(); name != "bedrock" {
		t.Errorf("ProviderName() = %q, want %q", name, "bedrock")
	}
}
