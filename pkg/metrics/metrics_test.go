package metrics_test

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scality/cosi-driver/pkg/metrics"
)

func TestGRPCFactorySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Test Suite")
}

type failingListener struct{}

func (f *failingListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("simulated listener failure")
}

func (f *failingListener) Close() error {
	return nil
}

func (f *failingListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
}

var _ = Describe("Metrics", func() {
	var (
		addr              string
		registry          *prometheus.Registry
		driverMetricsPath string
	)

	BeforeEach(func() {
		addr = "127.0.0.1:0" // Use a random available port
		registry = prometheus.NewRegistry()
		driverMetricsPath = "/metrics"
	})

	Describe("StartMetricsServerWithRegistry", func() {
		It("should start a metrics server successfully", func() {
			server, err := metrics.StartMetricsServerWithRegistry(addr, registry, driverMetricsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			resp, err := http.Get("http://" + server.Addr + driverMetricsPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			err = server.Close()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return an error when the address is invalid", func() {
			invalidAddr := "invalid:address"
			server, err := metrics.StartMetricsServerWithRegistry(invalidAddr, registry, driverMetricsPath)
			Expect(err).To(HaveOccurred())
			Expect(server).To(BeNil())
		})

		It("should log an error when the listener fails", func() {
			listener := &failingListener{}
			server := &http.Server{
				Handler: http.NewServeMux(),
			}

			go func() {
				err := server.Serve(listener)
				Expect(err).To(HaveOccurred())
			}()
		})
	})
})

var _ = Describe("InitializeMetrics", func() {
	var (
		registry          *prometheus.Registry
		driverMetricsPath string
	)

	BeforeEach(func() {
		registry = prometheus.NewRegistry()
		driverMetricsPath = "/metrics"
	})

	It("should serve metrics via an HTTP endpoint", func() {
		addr := "127.0.0.1:0"
		server, err := metrics.StartMetricsServerWithRegistry(addr, registry, driverMetricsPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(server).NotTo(BeNil())

		resp, err := http.Get("http://" + server.Addr + driverMetricsPath)
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		err = server.Close()
		Expect(err).NotTo(HaveOccurred())
	})
})
