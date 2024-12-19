package metrics

import (
	"net"
	"net/http"

	c "github.com/scality/cosi-driver/pkg/constants"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/klog/v2"
)

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
	prometheus.MustRegister(RequestsTotal)
}

func StartMetricsServer(addr string) (*http.Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return StartMetricsServerWithListener(listener)
}

func StartMetricsServerWithListener(listener net.Listener) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.Handle(c.MetricsPath, promhttp.Handler())

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
