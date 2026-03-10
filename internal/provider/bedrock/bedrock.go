package bedrock

import (
	"regexp"

	"github.com/faffige/llmwatcher/internal/provider"
	"github.com/faffige/llmwatcher/internal/provider/anthropic"
)

// Parser handles AWS Bedrock InvokeModel responses for Claude models.
// Bedrock uses the same response format as the Anthropic Messages API,
// so parsing is delegated to the anthropic package.
type Parser struct{}

func New() *Parser { return &Parser{} }

func (p *Parser) ProviderName() string { return "bedrock" }

// invokeRe matches Bedrock InvokeModel paths:
//
//	/model/{modelId}/invoke
var invokeRe = regexp.MustCompile(`^/model/[^/]+/invoke$`)

// invokeStreamRe matches Bedrock InvokeModelWithResponseStream paths:
//
//	/model/{modelId}/invoke-with-response-stream
var invokeStreamRe = regexp.MustCompile(`^/model/[^/]+/invoke-with-response-stream$`)

func (p *Parser) Parse(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	if method != "POST" || !invokeRe.MatchString(path) {
		return nil
	}
	return anthropic.ParseResponseBody("bedrock", method, path, statusCode, reqBody, respBody)
}

func (p *Parser) ParseStream(method, path string, statusCode int, reqBody, respBody []byte) *provider.CallRecord {
	if method != "POST" || !invokeStreamRe.MatchString(path) {
		return nil
	}
	return anthropic.ParseStreamBody("bedrock", method, path, statusCode, reqBody, respBody)
}
