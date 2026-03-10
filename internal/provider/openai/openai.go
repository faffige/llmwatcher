package openai

import (
	"encoding/json"
	"strings"

	"github.com/faffige/llmwatcher/internal/provider"
)

// Parser handles OpenAI chat completion responses (streaming and non-streaming).
type Parser struct{}

func New() *Parser { return &Parser{} }

func (p *Parser) ProviderName() string { return "openai" }

// chatCompletionResponse is the subset of the OpenAI response we care about.
type chatCompletionResponse struct {
	Model string `json:"model"`
	Usage *usage `json:"usage"`
	Error *apiError `json:"error"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (p *Parser) Parse(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	// Only handle chat completions for now.
	if !isChatCompletions(method, path) {
		return nil
	}

	rec := &provider.CallRecord{
		Provider:     "openai",
		Method:       method,
		Path:         path,
		Operation:    "chat",
		StatusCode:   statusCode,
		RequestBody:  reqBody,
		ResponseBody: respBody,
	}

	var resp chatCompletionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return rec
	}

	rec.Model = resp.Model
	if resp.Usage != nil {
		rec.InputTokens = resp.Usage.PromptTokens
		rec.OutputTokens = resp.Usage.CompletionTokens
		rec.TotalTokens = resp.Usage.TotalTokens
	}
	if resp.Error != nil {
		rec.ErrorType = resp.Error.Type
		rec.ErrorMsg = resp.Error.Message
	}

	return rec
}

func (p *Parser) ParseStream(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	return ParseStreaming(method, path, statusCode, reqBody, respBody)
}

func isChatCompletions(method, path string) bool {
	if method != "POST" {
		return false
	}
	cleaned := strings.TrimSuffix(path, "/")
	return cleaned == "/v1/chat/completions"
}
