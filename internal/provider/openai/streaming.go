package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"

	"github.com/faffige/llmwatcher/internal/provider"
)

// streamChunk is the subset of an OpenAI streaming chunk we care about.
type streamChunk struct {
	Model string       `json:"model"`
	Usage *usage       `json:"usage"`
	Error *apiError    `json:"error"`
}

// ParseStreaming parses a buffered SSE response body from an OpenAI
// streaming chat completion. It scans for the final chunk which contains
// the usage data (when stream_options.include_usage is true).
func ParseStreaming(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	if !isChatCompletions(method, path) {
		return nil
	}

	rec := &provider.CallRecord{
		Provider:     "openai",
		Method:       method,
		Path:         path,
		Operation:    "chat",
		StatusCode:   statusCode,
		IsStream:     true,
		RequestBody:  reqBody,
		ResponseBody: respBody,
	}

	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk streamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		// The model field appears in every chunk.
		if chunk.Model != "" {
			rec.Model = chunk.Model
		}

		// Usage only appears in the final chunk when include_usage is true.
		if chunk.Usage != nil {
			rec.InputTokens = chunk.Usage.PromptTokens
			rec.OutputTokens = chunk.Usage.CompletionTokens
			rec.TotalTokens = chunk.Usage.TotalTokens
		}

		if chunk.Error != nil {
			rec.ErrorType = chunk.Error.Type
			rec.ErrorMsg = chunk.Error.Message
		}
	}

	return rec
}
