package grpcfactory_test

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

var _ = Describe("gRPC Factory Server", func() {
	var (
		ctx               context.Context
		cancel            context.CancelFunc
		address           string
		identityServer    cosi.IdentityServer
		provisionerServer cosi.ProvisionerServer
		server            *grpcfactory.COSIProvisionerServer
	)

	BeforeEach(func() {
		// Generate a unique socket address for this test run
		address = generateUniqueAddress()

		ctx, cancel = context.WithCancel(context.Background())
		identityServer = &mockIdentityServer{}
		provisionerServer = &mockProvisionerServer{}
	})

	AfterEach(func() {
		cancel()
		// Clean up the socket file after each test
		os.Remove(address[7:])
	})

	Describe("Run", func() {
		It("should start the server and return no error", func() {
			var err error
			server, err = grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run(ctx)
			}()

			// Allow time for the server to start
			time.Sleep(100 * time.Millisecond)

			select {
			case err := <-errChan:
				Expect(err).NotTo(HaveOccurred())
			default:
				// No errors
			}
		})

		It("should return an error when reusing the same address", func() {
			// Use a fixed address to simulate reuse
			address := "unix:///tmp/test.sock"
			socketPath := address[7:] // Strip the `unix://` prefix

			// Start a stub listener on the address to occupy it
			listener, err := net.Listen("unix", socketPath)
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			// Try to start the gRPC server on the same address
			server2Ctx, server2Cancel := context.WithCancel(context.Background())
			defer server2Cancel()

			server2, err := grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server2).NotTo(BeNil())

			errChan2 := make(chan error, 1)
			go func() {
				errChan2 <- server2.Run(server2Ctx)
			}()

			// Expect the second server to fail immediately due to address reuse
			select {
			case err := <-errChan2:
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("address already in use"))
			case <-time.After(1 * time.Second):
				Fail("Expected an 'address already in use' error, but none was received")
			}

			// Clean up the socket file for future tests
			os.Remove(socketPath)
		})

		It("should handle unexpected shutdowns by canceling the context", func() {
			server, err := grpcfactory.NewCOSIProvisionerServer(address, identityServer, provisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run(ctx)
			}()

			// server startup time
			time.Sleep(100 * time.Millisecond)

			// Cancel the context to simulate an abrupt shutdown
			cancel()

			// Expect the Run function to exit cleanly or with context.Canceled error
			Eventually(errChan, 1*time.Second).Should(Receive(MatchError(context.Canceled)))
		})
	})
})
