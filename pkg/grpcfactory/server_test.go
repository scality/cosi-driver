package grpcfactory_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type mockIdentityServer struct {
	cosi.UnimplementedIdentityServer
}

type mockProvisionerServer struct {
	cosi.UnimplementedProvisionerServer
}

// generateUniqueAddress returns a unique Unix socket address for each test
func generateUniqueAddress() string {
	return fmt.Sprintf("unix:///tmp/test-%d.sock", time.Now().UnixNano())
}

type mockRegisterer struct{}

func (m *mockRegisterer) Register(prometheus.Collector) error {
	return fmt.Errorf("mock registration failure")
}

func (m *mockRegisterer) MustRegister(...prometheus.Collector) {
	panic("mock registration failure")
}

func (m *mockRegisterer) Unregister(prometheus.Collector) bool {
	return false
}

type failingListener struct{}

func (f *failingListener) Accept() (net.Conn, error) {
	return nil, fmt.Errorf("simulated listener failure")
}

func (f *failingListener) Close() error {
	return nil
}

func (f *failingListener) Addr() net.Addr {
	return &net.UnixAddr{Name: "mock", Net: "unix"}
}

var _ = Describe("gRPC Factory Server", Ordered, func() {
	var (
		address           string
		identityServer    cosi.IdentityServer
		provisionerServer cosi.ProvisionerServer
		server            *grpcfactory.COSIProvisionerServer
	)

	BeforeEach(func() {
		// Generate a unique socket address for this test run
		address = generateUniqueAddress()

		identityServer = &mockIdentityServer{}
		provisionerServer = &mockProvisionerServer{}
	})

	AfterEach(func() {
		socketPath := strings.TrimPrefix(address, "unix://")
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove socket file %s: %v\n", socketPath, err)
		}
	})

	Describe("Run", func() {
		It("should start the server and return no error", func(ctx SpecContext) {
			var err error
			server, err = grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			go func() {
				err := server.Run(ctx, prometheus.NewRegistry())
				if errors.Is(err, context.Canceled) {
					return // Expected when the context is canceled
				}
				Expect(err).NotTo(HaveOccurred())
			}()

			// Allow time for the server to start
			time.Sleep(100 * time.Millisecond)
		}, SpecTimeout(1*time.Second))

		It("should return an error when reusing the same address", func(ctx SpecContext) {
			socketPath := strings.TrimPrefix(address, "unix://")
			listener, err := net.Listen("unix", socketPath)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			server2, err := grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server2).NotTo(BeNil())

			// Run the second server and expect it to fail
			err = server2.Run(ctx, prometheus.NewRegistry())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("address already in use"))
		}, SpecTimeout(1*time.Second))

		It("should return an error for unsupported address schemes", func(ctx SpecContext) {
			invalidAddress := "http://invalid-scheme-address" // Address with an unsupported scheme

			server, err := grpcfactory.NewCOSIProvisionerServer(invalidAddress, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred()) // Ensure server creation succeeds
			Expect(server).NotTo(BeNil())

			// Wait for server.Run to return an error
			err = server.Run(ctx, prometheus.NewRegistry())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported scheme: expected 'unix'"))
		}, SpecTimeout(1*time.Second))

		It("should return an error when gRPC metrics registration fails", func(ctx SpecContext) {
			mockRegistry := &mockRegisterer{}

			server, err := grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			err = server.Run(ctx, mockRegistry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register gRPC metrics"))
		}, SpecTimeout(3*time.Second))

		It("should return an error when the address is invalid", func(ctx SpecContext) {
			// produce missing protocol scheme error
			invalidAddress := "::/invalid-address"

			server, err := grpcfactory.NewCOSIProvisionerServer(invalidAddress, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			err = server.Run(ctx, prometheus.NewRegistry())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("missing protocol scheme"))
		}, SpecTimeout(1*time.Second))
	})
})
