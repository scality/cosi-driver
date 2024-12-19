package metrics_test

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/metrics"

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

		// Create and register gRPC metrics with the fresh registry
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

		// Increment the cosi_requests_total counter
		Expect(registry.Register(metrics.RequestsTotal)).To(Succeed())
		metrics.RequestsTotal.With(prometheus.Labels{"method": "GET", "status": "200"}).Inc()

		// Wrap the Prometheus handler with the fresh registry
		mux := http.NewServeMux()
		mux.Handle(constants.MetricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

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
		resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, constants.MetricsPath))
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
		Expect(metricsOutput).To(ContainSubstring(`cosi_requests_total{method="GET",status="200"} 1`))
	})

	It("should gracefully shutdown the server", func() {
		Expect(server.Close()).To(Succeed())
	})

	It("should handle invalid metric labels gracefully", func() {
		Expect(func() {
			grpcMetrics.WithLabelValues("InvalidMethod", "InvalidService", "InvalidType").Inc()
		}).NotTo(Panic())
	})

	It("should return 404 for invalid metrics paths", func() {
		addr := listener.Addr().String()
		resp, err := http.Get(fmt.Sprintf("http://%s/invalid-path", addr))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		Expect(resp.Body.Close()).To(Succeed())
	})

	It("should handle multiple concurrent requests", func() {
		addr := listener.Addr().String()
		const numRequests = 10
		done := make(chan bool, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, constants.MetricsPath))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
				Expect(resp.Body.Close()).To(Succeed())
				done <- true
			}()
		}

		for i := 0; i < numRequests; i++ {
			<-done
		}
	})

	It("should expose additional custom metrics", func() {
		customCounter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: "custom_metric_total",
			Help: "A custom metric for testing purposes.",
		})
		Expect(registry.Register(customCounter)).To(Succeed())
		customCounter.Add(42)

		addr := listener.Addr().String()
		resp, err := http.Get(fmt.Sprintf("http://%s%s", addr, constants.MetricsPath))
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		// Validate custom metrics
		body, err := io.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Body.Close()).To(Succeed())
		Expect(string(body)).To(ContainSubstring(`custom_metric_total 42`))
	})
})

var _ = Describe("Metrics Server Functions", func() {
	var listener net.Listener
	var addr string

	AfterEach(func() {
		// Ensure the listener is closed after each test
		if listener != nil {
			_ = listener.Close()
		}
	})

	Describe("StartMetricsServer", func() {
		It("should start the server successfully with a valid address", func() {
			// Start the server
			addr = "127.0.0.1:0" // Dynamic port
			server, err := metrics.StartMetricsServer(addr)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			// Use the actual address from the server
			resp, err := http.Get(fmt.Sprintf("http://%s%s", server.Addr, constants.MetricsPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Body.Close()).To(Succeed())

			// Clean up
			_ = server.Close()
		})

		It("should return an error when failing to listen on the address", func() {
			addr = "invalid_address"
			server, err := metrics.StartMetricsServer(addr)
			Expect(err).To(HaveOccurred())
			Expect(server).To(BeNil())
		})
	})

	Describe("StartMetricsServerWithListener", func() {
		It("should start the server successfully with a valid listener", func() {
			// Create a valid listener
			var err error
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())

			server, err := metrics.StartMetricsServerWithListener(listener)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			resp, err := http.Get(fmt.Sprintf("http://%s%s", listener.Addr().String(), constants.MetricsPath))
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(resp.Body.Close()).To(Succeed())

			// Clean up
			_ = server.Close()
		})

		It("should handle errors during server operation gracefully", func() {
			// Create a valid listener
			var err error
			listener, err = net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())

			// Start the server
			server, err := metrics.StartMetricsServerWithListener(listener)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			// Close the listener to simulate an error during `srv.Serve`
			_ = listener.Close()

			// Wait briefly to ensure the goroutine handling `srv.Serve` runs
			time.Sleep(100 * time.Millisecond)

			// Clean up
			_ = server.Close()
		})
	})
})
