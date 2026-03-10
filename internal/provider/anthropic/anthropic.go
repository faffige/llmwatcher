package anthropic

import (
	"encoding/json"
	"strings"

	"github.com/faffige/llmwatcher/internal/provider"
)

// Parser handles Anthropic Messages API responses (streaming and non-streaming).
type Parser struct{}

func New() *Parser { return &Parser{} }

func (p *Parser) ProviderName() string { return "anthropic" }

// messagesResponse is the subset of the Anthropic response we care about.
type messagesResponse struct {
	Model     string    `json:"model"`
	Type      string    `json:"type"`
	StopReason string   `json:"stop_reason"`
	Usage     *usage    `json:"usage"`
	Error     *apiError `json:"error"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// errorWrapper handles Anthropic's top-level error format:
// {"type": "error", "error": {"type": "...", "message": "..."}}
type errorWrapper struct {
	Type  string    `json:"type"`
	Error *apiError `json:"error"`
}

func (p *Parser) Parse(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	if !isMessages(method, path) {
		return nil
	}

	rec := &provider.CallRecord{
		Provider:     "anthropic",
		Method:       method,
		Path:         path,
		Operation:    "chat",
		StatusCode:   statusCode,
		RequestBody:  reqBody,
		ResponseBody: respBody,
	}

	// Try error wrapper first (Anthropic returns {"type":"error","error":{...}} on failures).
	var ew errorWrapper
	if err := json.Unmarshal(respBody, &ew); err != nil {
		return rec
	}
	if ew.Type == "error" && ew.Error != nil {
		rec.ErrorType = ew.Error.Type
		rec.ErrorMsg = ew.Error.Message
		return rec
	}

	var resp messagesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return rec
	}

	rec.Model = resp.Model
	if resp.Usage != nil {
		rec.InputTokens = resp.Usage.InputTokens
		rec.OutputTokens = resp.Usage.OutputTokens
		rec.TotalTokens = resp.Usage.InputTokens + resp.Usage.OutputTokens
	}

	return rec
}

func (p *Parser) ParseStream(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	return parseStreaming(method, path, statusCode, reqBody, respBody)
}

func isMessages(method, path string) bool {
	if method != "POST" {
		return false
	}
	cleaned := strings.TrimSuffix(path, "/")
	return cleaned == "/v1/messages"
}
