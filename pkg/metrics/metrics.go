package metrics

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

// StartMetricsServerWithRegistry starts an HTTP metrics server with a custom Prometheus registry.
func StartMetricsServerWithRegistry(addr string, registry prometheus.Gatherer, driverMetricsPath string) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return StartMetricsServerWithListenerAndRegistry(listener, registry, driverMetricsPath)
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
