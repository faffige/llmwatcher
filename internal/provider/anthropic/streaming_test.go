package anthropic

import (
	"testing"
)

const sampleSSE = `event: message_start
data: {"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","model":"claude-sonnet-4-20250514","content":[],"stop_reason":null,"usage":{"input_tokens":12,"output_tokens":0}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"!"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}

event: message_stop
data: {"type":"message_stop"}
`

func TestParseStreaming_WithUsage(t *testing.T) {
	rec := parseStreaming("POST", "/v1/messages", 200, nil, []byte(sampleSSE))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}

	if rec.Provider != "anthropic" {
		t.Errorf("provider = %q, want %q", rec.Provider, "anthropic")
	}
	if rec.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want %q", rec.Model, "claude-sonnet-4-20250514")
	}
	if !rec.IsStream {
		t.Error("expected IsStream = true")
	}
	if rec.Operation != "chat" {
		t.Errorf("operation = %q, want %q", rec.Operation, "chat")
	}
	if rec.InputTokens != 12 {
		t.Errorf("input_tokens = %d, want 12", rec.InputTokens)
	}
	if rec.OutputTokens != 5 {
		t.Errorf("output_tokens = %d, want 5", rec.OutputTokens)
	}
	if rec.TotalTokens != 17 {
		t.Errorf("total_tokens = %d, want 17", rec.TotalTokens)
	}
}

func TestParseStreaming_NonMessagesEndpoint(t *testing.T) {
	rec := parseStreaming("POST", "/v1/complete", 200, nil, nil)
	if rec != nil {
		t.Errorf("expected nil for non-messages endpoint, got %+v", rec)
	}
}

func TestParseStreaming_ErrorInStream(t *testing.T) {
	sse := `event: error
data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}
`
	rec := parseStreaming("POST", "/v1/messages", 529, nil, []byte(sse))
	if rec == nil {
		t.Fatal("expected a record, got nil")
	}
	if rec.ErrorType != "overloaded_error" {
		t.Errorf("error_type = %q, want %q", rec.ErrorType, "overloaded_error")
	}
	if rec.ErrorMsg != "Overloaded" {
		t.Errorf("error_msg = %q, want %q", rec.ErrorMsg, "Overloaded")
	}
}

func TestParseStreaming_EmptyBody(t *testing.T) {
	rec := parseStreaming("POST", "/v1/messages", 200, nil, []byte{})
	if rec == nil {
		t.Fatal("expected a record even with empty body")
	}
	if rec.Model != "" {
		t.Errorf("model = %q, want empty", rec.Model)
	}
}
