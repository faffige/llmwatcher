package openai

import (
	"testing"
)

const sampleSSE = `data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","model":"gpt-4o-2024-08-06","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","model":"gpt-4o-2024-08-06","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","model":"gpt-4o-2024-08-06","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}

data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","model":"gpt-4o-2024-08-06","choices":[],"usage":{"prompt_tokens":8,"completion_tokens":2,"total_tokens":10}}

data: [DONE]
`

func TestParseStreaming_WithUsage(t *testing.T) {
	rec := ParseStreaming("POST", "/v1/chat/completions", 200, nil, []byte(sampleSSE))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "openai" {
		t.Errorf("provider = %q, want %q", rec.Provider, "openai")
	}
	if rec.Model != "gpt-4o-2024-08-06" {
		t.Errorf("model = %q, want %q", rec.Model, "gpt-4o-2024-08-06")
	}
	if !rec.IsStream {
		t.Error("expected IsStream = true")
	}
	if rec.Operation != "chat" {
		t.Errorf("operation = %q, want %q", rec.Operation, "chat")
	}
	if rec.InputTokens != 8 {
		t.Errorf("input_tokens = %d, want 8", rec.InputTokens)
	}
	if rec.OutputTokens != 2 {
		t.Errorf("output_tokens = %d, want 2", rec.OutputTokens)
	}
	if rec.TotalTokens != 10 {
		t.Errorf("total_tokens = %d, want 10", rec.TotalTokens)
	}
}

func TestParseStreaming_WithoutUsage(t *testing.T) {
	// Stream without include_usage — no usage chunk.
	sse := `data: {"id":"chatcmpl-abc","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hi"},"finish_reason":"stop"}]}

data: [DONE]
`
	rec := ParseStreaming("POST", "/v1/chat/completions", 200, nil, []byte(sse))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}
	if rec.Model != "gpt-4o" {
		t.Errorf("model = %q, want %q", rec.Model, "gpt-4o")
	}
	if rec.TotalTokens != 0 {
		t.Errorf("total_tokens = %d, want 0 (no usage chunk)", rec.TotalTokens)
	}
}

func TestParseStreaming_NonChatEndpoint(t *testing.T) {
	rec := ParseStreaming("POST", "/v1/embeddings", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-chat endpoint, got %+v", rec)
	}
}

func TestParseStreaming_ErrorInStream(t *testing.T) {
	sse := `data: {"error":{"type":"server_error","message":"internal error"}}

data: [DONE]
`
	rec := ParseStreaming("POST", "/v1/chat/completions", 500, nil, []byte(sse))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}
	if rec.ErrorType != "server_error" {
		t.Errorf("error_type = %q, want %q", rec.ErrorType, "server_error")
	}
	if rec.ErrorMsg != "internal error" {
		t.Errorf("error_msg = %q, want %q", rec.ErrorMsg, "internal error")
	}
}

func TestParseStreaming_EmptyBody(t *testing.T) {
	rec := ParseStreaming("POST", "/v1/chat/completions", 200, nil, []byte{})
	if rec == nil {
		t.Fatal("expected a record even with empty body")
	}
	if rec.Model != "" {
		t.Errorf("model = %q, want empty", rec.Model)
	}
}
