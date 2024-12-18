package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

// Define custom metrics if needed
var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cosi_requests_total",
			Help: "Total number of requests handled by the COSI driver.",
		},
		[]string{"method", "status"},
	)
)

func init() {
	// Register your custom metrics here
	prometheus.MustRegister(RequestsTotal)
}

// StartMetricsServer starts the HTTP server that serves Prometheus metrics
func StartMetricsServer(addr string) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		klog.InfoS("Starting Prometheus metrics server", "address", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			klog.ErrorS(err, "Failed to start metrics server")
		}
	}()

	return srv, nil
}
