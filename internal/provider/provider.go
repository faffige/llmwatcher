package provider

import "time"

// CallRecord holds the parsed metadata from a single LLM API call.
type CallRecord struct {
	// Identity
	ID       string // ULID, assigned by pipeline
	Provider string
	Model    string

	// Timing
	StartedAt  time.Time
	DurationMs int64

	// HTTP
	Method     string
	Path       string
	Operation  string // "chat", "completion", "embedding"
	StatusCode int
	IsStream   bool

	// Token usage (as reported by the provider)
	InputTokens  int
	OutputTokens int
	TotalTokens  int

	// Error info (populated on non-2xx responses)
	ErrorType string
	ErrorMsg  string

	// Request/response bodies (opt-in, can be nil)
	RequestBody  []byte
	ResponseBody []byte
}

// Parser extracts a CallRecord from a captured request/response pair.
type Parser interface {
	// ProviderName returns the canonical name (e.g. "openai").
	ProviderName() string

	// Parse inspects the request and response bodies and populates a CallRecord.
	// It returns nil if the request is not one it can parse (e.g. unknown endpoint).
	Parse(method, path string, statusCode int, reqBody, respBody []byte) *CallRecord

	// ParseStream parses a buffered SSE streaming response.
	// It returns nil if the request is not one it can parse.
	ParseStream(method, path string, statusCode int, reqBody, respBody []byte) *CallRecord
}
