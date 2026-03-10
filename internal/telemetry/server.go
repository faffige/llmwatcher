package telemetry

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewMetricsServer creates an HTTP server that serves the Prometheus /metrics
// endpoint on the given address (e.g. "127.0.0.1:9090").
func NewMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
