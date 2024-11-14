package grpcfactory_test

import (
	"context"
	"net"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type MockIdentityServer struct {
	cosi.UnimplementedIdentityServer
}

type MockProvisionerServer struct {
	cosi.UnimplementedProvisionerServer
}

var _ = Describe("gRPC Factory Client", func() {
	var (
		client     *grpcfactory.COSIProvisionerClient
		grpcServer *grpc.Server
		listener   net.Listener
		address    string
	)

	BeforeEach(func() {
		address = "unix:///tmp/test.sock"
		grpcServer = grpc.NewServer()

		// Remove any existing socket file to avoid "address already in use" errors
		_ = os.Remove(address[7:])

		// Create the listener
		var err error
		listener, err = net.Listen("unix", address[7:])
		Expect(err).NotTo(HaveOccurred(), "Failed to create Unix listener for gRPC server")

		// Register mock servers
		cosi.RegisterIdentityServer(grpcServer, &MockIdentityServer{})
		cosi.RegisterProvisionerServer(grpcServer, &MockProvisionerServer{})

		// Start the gRPC server in a separate goroutine
		go func() {
			err := grpcServer.Serve(listener)
			if err != nil && err != grpc.ErrServerStopped {
				GinkgoWriter.Println("gRPC server encountered an error:", err)
			}
		}()
	})

	AfterEach(func() {
		// Stop the gRPC server and close the listener
		grpcServer.Stop()
		if listener != nil {
			listener.Close()
		}
		// Remove the Unix socket file to clean up after each test
		_ = os.Remove(address[7:])
	})

	Describe("Initialization and Connection", func() {
		It("should initialize and connect COSIProvisionerClient", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Add insecure credentials to the dial options for Unix socket
			dialOpts := []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
			}

			var err error
			client, err = grpcfactory.NewCOSIProvisionerClient(ctx, address, dialOpts, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
			Expect(client.IdentityClient).NotTo(BeNil())
			Expect(client.ProvisionerClient).NotTo(BeNil())
		})
	})

	Describe("Interface Implementation", func() {
		It("should implement cosi.IdentityClient and cosi.ProvisionerClient interfaces", func() {
			client = &grpcfactory.COSIProvisionerClient{
				IdentityClient:    cosi.NewIdentityClient(nil),
				ProvisionerClient: cosi.NewProvisionerClient(nil),
			}

			var _ cosi.IdentityClient = client
			var _ cosi.ProvisionerClient = client
		})
	})

	Describe("Interceptor Usage", func() {
		It("should use ApiLogger as an interceptor if debug is true", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			client, err := grpcfactory.NewDefaultCOSIProvisionerClient(ctx, address, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should initialize without interceptors if debug is false", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			client, err := grpcfactory.NewDefaultCOSIProvisionerClient(ctx, address, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})
	})

	Describe("Error Handling", func() {
		It("should return an error if given an invalid address", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Attempt to connect using an invalid address format
			_, err := grpcfactory.NewCOSIProvisionerClient(ctx, "invalid-address", nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unsupported scheme"))
		})
	})
})
