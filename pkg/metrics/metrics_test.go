package metrics_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func TestGRPCFactorySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Test Suite")
}

var _ = Describe("Metrics Server", func() {
	var (
		server      *http.Server
		listener    net.Listener
		registry    *prometheus.Registry
		grpcMetrics *prometheus.CounterVec
	)

	BeforeEach(func() {
		// Create a fresh registry
		registry = prometheus.NewRegistry()

		grpcMetrics = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "grpc_server_started_total",
				Help: "Total number of RPCs started on the server.",
			},
			[]string{"grpc_method", "grpc_service", "grpc_type"},
		)
		Expect(registry.Register(grpcMetrics)).To(Succeed())

		// Increment gRPC metrics to simulate usage
		grpcMetrics.WithLabelValues("DriverGetInfo", "cosi.v1alpha1.Identity", "unary").Add(5)
		grpcMetrics.WithLabelValues("DriverCreateBucket", "cosi.v1alpha1.Provisioner", "unary").Add(4)
		grpcMetrics.WithLabelValues("DriverDeleteBucket", "cosi.v1alpha1.Provisioner", "unary").Add(3)
		grpcMetrics.WithLabelValues("DriverGrantBucketAccess", "cosi.v1alpha1.Provisioner", "unary").Add(2)
		grpcMetrics.WithLabelValues("DriverRevokeBucketAccess", "cosi.v1alpha1.Provisioner", "unary").Add(1)

		// Wrap the Prometheus handler with the fresh registry
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

		// Create and start the metrics server
		var err error
		server = &http.Server{Handler: mux}
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		go func() {
			err := server.Serve(listener)
			if err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()

		// Wait for the server to start
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		// Shutdown the server if it is still running
		if server != nil {
			err := server.Close()
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				Expect(err).NotTo(HaveOccurred())
			}
		}

		// Close the listener if it's still open
		if listener != nil {
			err := listener.Close()
			if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
				Expect(err).NotTo(HaveOccurred())
			}
		}
	})

	It("should expose gRPC metrics on the Prometheus endpoint", func() {
		addr := listener.Addr().String()
		resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, "/metrics"))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// Read and validate metrics
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Body.Close()).To(Succeed())
		metricsOutput := string(body)

		Expect(metricsOutput).To(ContainSubstring(`grpc_server_started_total{grpc_method="DriverGetInfo",grpc_service="cosi.v1alpha1.Identity",grpc_type="unary"} 5`))
		Expect(metricsOutput).To(ContainSubstring(`grpc_server_started_total{grpc_method="DriverCreateBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 4`))
		Expect(metricsOutput).To(ContainSubstring(`grpc_server_started_total{grpc_method="DriverDeleteBucket",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 3`))
		Expect(metricsOutput).To(ContainSubstring(`grpc_server_started_total{grpc_method="DriverGrantBucketAccess",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 2`))
		Expect(metricsOutput).To(ContainSubstring(`grpc_server_started_total{grpc_method="DriverRevokeBucketAccess",grpc_service="cosi.v1alpha1.Provisioner",grpc_type="unary"} 1`))
	})
})
