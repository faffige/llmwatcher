package proxy

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/faffige/llmwatcher/internal/config"
	"github.com/faffige/llmwatcher/internal/provider"
)

// Server is the reverse proxy server that routes LLM API calls
// to the appropriate upstream provider.
type Server struct {
	mux    *http.ServeMux
	logger *slog.Logger
}

// New creates a proxy server with routes for each enabled provider.
// Parsers is a map of provider name → Parser; if a parser exists for a
// provider, the recorder middleware will capture and parse requests.
// Transports is an optional map of provider name → http.RoundTripper for
// providers that need custom transport behaviour (e.g. AWS Sig V4 signing).
func New(cfg *config.Config, parsers map[string]provider.Parser, sink RecordSink, logger *slog.Logger, transports ...map[string]http.RoundTripper) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		logger: logger,
	}

	var transportMap map[string]http.RoundTripper
	if len(transports) > 0 {
		transportMap = transports[0]
	}

	for name, provCfg := range cfg.Providers {
		if !provCfg.Enabled {
			logger.Info("provider disabled, skipping", "provider", name)
			continue
		}

		upstream, err := url.Parse(provCfg.Upstream)
		if err != nil {
			logger.Error("invalid upstream URL", "provider", name, "upstream", provCfg.Upstream, "error", err)
			continue
		}

		prefix := fmt.Sprintf("/v1/%s/", name)
		rp := newReverseProxy(upstream, prefix, logger, transportMap[name])

		// Wrap with recorder middleware if we have a parser for this provider.
		handler := newRecorder(rp, parsers[name], sink, logger)

		s.mux.Handle(prefix, handler)
		logger.Info("registered provider route", "provider", name, "prefix", prefix, "upstream", upstream.String())
	}

	s.mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	return s
}

// Handler returns the HTTP handler for the proxy server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

func newReverseProxy(upstream *url.URL, prefix string, logger *slog.Logger, transport http.RoundTripper) http.Handler {
	rp := &httputil.ReverseProxy{
		// FlushInterval -1 means flush immediately — required for SSE streaming.
		FlushInterval: -1,
		Transport:     transport, // nil means http.DefaultTransport
		Rewrite: func(r *httputil.ProxyRequest) {
			// Strip the llmwatcher prefix and forward to upstream.
			// e.g. /v1/openai/v1/chat/completions → https://api.openai.com/v1/chat/completions
			origPath := r.In.URL.Path
			stripped := strings.TrimPrefix(origPath, strings.TrimSuffix(prefix, "/"))
			r.Out.URL.Scheme = upstream.Scheme
			r.Out.URL.Host = upstream.Host
			r.Out.URL.Path = stripped
			r.Out.URL.RawQuery = r.In.URL.RawQuery
			r.Out.Host = upstream.Host

			// Copy all headers from the original request (including auth).
			r.Out.Header = r.In.Header.Clone()

			logger.Debug("proxying request",
				"method", r.In.Method,
				"from", origPath,
				"to", r.Out.URL.String(),
			)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			logger.Error("proxy error", "url", r.URL.String(), "error", err)
			http.Error(w, fmt.Sprintf("llmwatcher proxy error: %v", err), http.StatusBadGateway)
		},
	}

	return rp
}
