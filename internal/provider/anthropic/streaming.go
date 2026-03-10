package anthropic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"

	"github.com/faffige/llmwatcher/internal/provider"
)

// Anthropic SSE streaming uses typed events:
//   event: message_start   → {"type":"message_start","message":{"model":"...","usage":{"input_tokens":N}}}
//   event: content_block_delta → content chunks
//   event: message_delta   → {"type":"message_delta","usage":{"output_tokens":N}}
//   event: message_stop    → end of stream

// messageStartEvent is the payload for "event: message_start".
type messageStartEvent struct {
	Type    string               `json:"type"`
	Message *messageStartMessage `json:"message"`
}

type messageStartMessage struct {
	Model string `json:"model"`
	Usage *usage `json:"usage"`
}

// messageDeltaEvent is the payload for "event: message_delta".
type messageDeltaEvent struct {
	Type  string `json:"type"`
	Usage *struct {
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// streamErrorEvent captures errors in streaming responses.
type streamErrorEvent struct {
	Type  string    `json:"type"`
	Error *apiError `json:"error"`
}

func parseStreaming(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	if !isMessages(method, path) {
		return nil
	}

	rec := &provider.CallRecord{
		Provider:     "anthropic",
		Method:       method,
		Path:         path,
		Operation:    "chat",
		StatusCode:   statusCode,
		IsStream:     true,
		RequestBody:  reqBody,
		ResponseBody: respBody,
	}

	scanner := bufio.NewScanner(bytes.NewReader(respBody))
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		// Track the event type.
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		switch currentEvent {
		case "message_start":
			var evt messageStartEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			if evt.Message != nil {
				rec.Model = evt.Message.Model
				if evt.Message.Usage != nil {
					rec.InputTokens = evt.Message.Usage.InputTokens
				}
			}

		case "message_delta":
			var evt messageDeltaEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			if evt.Usage != nil {
				rec.OutputTokens = evt.Usage.OutputTokens
				rec.TotalTokens = rec.InputTokens + rec.OutputTokens
			}

		case "error":
			var evt streamErrorEvent
			if err := json.Unmarshal([]byte(data), &evt); err != nil {
				continue
			}
			if evt.Error != nil {
				rec.ErrorType = evt.Error.Type
				rec.ErrorMsg = evt.Error.Message
			}
		}
	}

	// Calculate total if we have both parts.
	if rec.TotalTokens == 0 && (rec.InputTokens > 0 || rec.OutputTokens > 0) {
		rec.TotalTokens = rec.InputTokens + rec.OutputTokens
	}

	return rec
}
