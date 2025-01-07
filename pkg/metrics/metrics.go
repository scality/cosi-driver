package metrics

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

var (
	S3RequestsTotal    *prometheus.CounterVec
	S3RequestDuration  *prometheus.HistogramVec
	IAMRequestsTotal   *prometheus.CounterVec
	IAMRequestDuration *prometheus.HistogramVec
)

// StartMetricsServerWithRegistry starts an HTTP server for exposing metrics using a custom registry.
func StartMetricsServerWithRegistry(addr string, registry prometheus.Gatherer, metricsPath string) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.Handle(metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))

	srv := &http.Server{
		Handler: mux,
		Addr:    listener.Addr().String(),
	}

	go func() {
		klog.InfoS("Starting Prometheus metrics server", "address", listener.Addr().String())
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			klog.ErrorS(err, "Failed to start metrics server")
		}
	}()

	return srv, nil
}
