package grpcfactory_test

import (
	"context"
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

var _ = Describe("gRPC Factory Server", Ordered, func() {
	var (
		address           string
		identityServer    cosi.IdentityServer
		provisionerServer cosi.ProvisionerServer
		server            *grpcfactory.COSIProvisionerServer
		registry          *prometheus.Registry
	)

	BeforeEach(func() {
		// Generate a unique socket address for this test run
		address = generateUniqueAddress()

		identityServer = &mockIdentityServer{}
		provisionerServer = &mockProvisionerServer{}

		// Create a custom Prometheus registry for this test
		registry = prometheus.NewRegistry()
	})

	AfterEach(func() {
		// Clean up the Unix socket file
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

			runErrChan := make(chan error)
			go func() {
				defer GinkgoRecover()
				runErrChan <- server.Run(ctx, registry) // Pass registry here for metrics registration
			}()

			time.Sleep(100 * time.Millisecond)

			ctxCancel, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			<-ctxCancel.Done()

			Expect(<-runErrChan).To(SatisfyAny(BeNil(), Equal(context.Canceled)))
		}, SpecTimeout(2*time.Second))

		It("should return an error when reusing the same address", func(ctx SpecContext) {
			// Manually create a listener to occupy the socket
			socketPath := strings.TrimPrefix(address, "unix://")
			listener, err := net.Listen("unix", socketPath)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			// Pass nil instead of registry for the ServerOptions parameter
			server, err = grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			// Run the server with the registry
			err = server.Run(ctx, registry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("address already in use"))
		}, SpecTimeout(1*time.Second))

		It("should return an error for unsupported address schemes", func(ctx SpecContext) {
			var (
				server *grpcfactory.COSIProvisionerServer
				err    error
			)

			invalidAddress := "http://invalid-scheme-address"

			server, err = grpcfactory.NewCOSIProvisionerServer(invalidAddress, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred()) // Ensure server creation succeeds
			Expect(server).NotTo(BeNil())

			// Attempt to run the server with the registry
			err = server.Run(ctx, prometheus.NewRegistry()) // Pass a custom registry here
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported scheme: expected 'unix'"))
		}, SpecTimeout(1*time.Second))
	})
})
