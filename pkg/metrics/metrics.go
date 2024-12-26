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

// InitializeMetrics initializes the metrics with a given prefix and registers them to a registry.
func InitializeMetrics(prefix string, registry prometheus.Registerer) {
	S3RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "s3_requests_total",
			Help:      "Total number of S3 requests, categorized by method and status.",
		},
		[]string{"method", "status", "trace_id"},
	)

	S3RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: prefix,
			Name:      "s3_request_duration_seconds",
			Help:      "Duration of S3 requests in seconds, categorized by method and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status", "trace_id"},
	)

	IAMRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "iam_requests_total",
			Help:      "Total number of IAM requests, categorized by method and status.",
		},
		[]string{"method", "status", "trace_id"},
	)

	IAMRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: prefix,
			Name:      "iam_request_duration_seconds",
			Help:      "Duration of IAM requests in seconds, categorized by method and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status", "trace_id"},
	)

	registry.MustRegister(S3RequestsTotal, S3RequestDuration, IAMRequestsTotal, IAMRequestDuration)

	klog.InfoS("Custom metrics initialized", "prefix", prefix)
}

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
