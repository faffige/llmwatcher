package proxy

import (
	"bytes"
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
		r.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	// Wrap the response writer to capture status and body.
	rw := &responseRecorder{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
		statusCode:     http.StatusOK,
	}

	rec.next.ServeHTTP(rw, r)

	duration := time.Since(start)

	// Parse asynchronously to avoid adding latency to the response.
	go rec.parse(r.Method, r.URL.Path, rw.statusCode, reqBody, rw.body.Bytes(), start, duration)
}

func (rec *recorder) parse(method, path string, statusCode int, reqBody, respBody []byte, start time.Time, duration time.Duration) {
	record := rec.parser.Parse(method, path, statusCode, reqBody, respBody)
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
		"input_tokens", record.InputTokens,
		"output_tokens", record.OutputTokens,
		"total_tokens", record.TotalTokens,
		"duration_ms", record.DurationMs,
	)

	if rec.sink != nil {
		rec.sink(record)
	}
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
