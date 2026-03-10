package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/faffige/llmwatcher/internal/provider"
)

// RecordSink receives parsed CallRecords for async processing.
type RecordSink func(rec *provider.CallRecord)

// recorder is middleware that captures request/response bodies,
// passes them through a provider.Parser, and submits the resulting
// CallRecord to a sink (typically the pipeline).
type recorder struct {
	next   http.Handler
	parser provider.Parser
	sink   RecordSink
	logger *slog.Logger
}

func newRecorder(next http.Handler, parser provider.Parser, sink RecordSink, logger *slog.Logger) http.Handler {
	if parser == nil {
		return next
	}
	return &recorder{next: next, parser: parser, sink: sink, logger: logger}
}

func (rec *recorder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Capture request body.
	var reqBody []byte
	if r.Body != nil {
		reqBody, _ = io.ReadAll(r.Body)
		r.Body.Close()
	}

	// Detect streaming and allow the provider to modify the request if needed.
	isStream := isStreamingRequest(reqBody)
	if isStream {
		if mod, ok := rec.parser.(provider.RequestModifier); ok {
			reqBody = mod.ModifyStreamingRequest(reqBody)
		}
	}

	r.Body = io.NopCloser(bytes.NewReader(reqBody))
	r.ContentLength = int64(len(reqBody))

	// Wrap the response writer to capture status and body.
	rw := &responseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}

	rec.next.ServeHTTP(rw, r)

	duration := time.Since(start)

	// Parse asynchronously to avoid adding latency to the response.
	go rec.parse(r.Method, r.URL.Path, rw.statusCode, isStream, reqBody, rw.body.Bytes(), start, duration)
}

func (rec *recorder) parse(method, path string, statusCode int, isStream bool, reqBody, respBody []byte, start time.Time, duration time.Duration) {
	var record *provider.CallRecord
	if isStream {
		record = rec.parser.ParseStream(method, path, statusCode, reqBody, respBody)
	} else {
		record = rec.parser.Parse(method, path, statusCode, reqBody, respBody)
	}
	if record == nil {
		return
	}

	record.StartedAt = start
	record.DurationMs = duration.Milliseconds()

	rec.logger.Info("request recorded",
		"provider", record.Provider,
		"model", record.Model,
		"method", record.Method,
		"path", record.Path,
		"status", record.StatusCode,
		"stream", record.IsStream,
		"input_tokens", record.InputTokens,
		"output_tokens", record.OutputTokens,
		"total_tokens", record.TotalTokens,
		"duration_ms", record.DurationMs,
	)

	if rec.sink != nil {
		rec.sink(record)
	}
}

// isStreamingRequest checks if the JSON request body has "stream": true.
func isStreamingRequest(body []byte) bool {
	if len(body) == 0 {
		return false
	}
	var req struct {
		Stream bool `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return false
	}
	return req.Stream
}

// responseRecorder wraps http.ResponseWriter to capture the status code and body.
type responseRecorder struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	wroteHead  bool
}

func (rw *responseRecorder) WriteHeader(code int) {
	if !rw.wroteHead {
		rw.statusCode = code
		rw.wroteHead = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseRecorder) Write(b []byte) (int, error) {
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

// Flush supports streaming responses.
func (rw *responseRecorder) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
