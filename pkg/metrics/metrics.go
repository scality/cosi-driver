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

// StartMetricsServerWithRegistry starts an HTTP metrics server with a custom Prometheus registry.
func StartMetricsServerWithRegistry(addr string, registry prometheus.Gatherer, driverMetricsPath string, driverMetricsPrefix string) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	// Initialize metrics with the provided prefix
	InitializeMetrics(driverMetricsPrefix)

	// Register the initialized metrics with the Prometheus registry
	prometheus.MustRegister(
		S3RequestsTotal,
		S3RequestDuration,
		IAMRequestsTotal,
		IAMRequestDuration,
	)

	return StartMetricsServerWithListenerAndRegistry(listener, registry, driverMetricsPath)
}

// initializeMetrics dynamically initializes metrics with the given prefix.
func InitializeMetrics(prefix string) {
	S3RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "s3_requests_total",
			Help:      "Total number of S3 requests, categorized by method and status.",
		},
		[]string{"method", "status"},
	)

	S3RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: prefix,
			Name:      "s3_request_duration_seconds",
			Help:      "Duration of S3 requests in seconds, categorized by method and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	IAMRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: prefix,
			Name:      "iam_requests_total",
			Help:      "Total number of IAM requests, categorized by method and status.",
		},
		[]string{"method", "status"},
	)

	IAMRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: prefix,
			Name:      "iam_request_duration_seconds",
			Help:      "Duration of IAM requests in seconds, categorized by method and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	klog.InfoS("Custom metrics initialized", "prefix", prefix)
}

// StartMetricsServerWithListenerAndRegistry starts an HTTP server with a custom registry, listener on specified driver metrics path.
func StartMetricsServerWithListenerAndRegistry(listener net.Listener, registry prometheus.Gatherer, driverMetricsPath string) (*http.Server, error) {
	mux := http.NewServeMux()

	mux.Handle(driverMetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{
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
