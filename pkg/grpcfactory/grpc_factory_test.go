package grpcfactory_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

var _ = Describe("gRPC Factory Provisioner", func() {
	var (
		address               string
		mockIdentityServer    cosi.IdentityServer
		mockProvisionerServer cosi.ProvisionerServer
	)

	BeforeEach(func() {
		address = "unix:///tmp/test.sock"
		mockIdentityServer = &MockIdentityServer{}
		mockProvisionerServer = &MockProvisionerServer{}
		_ = os.Remove(address[7:])
	})

	AfterEach(func() {
		_ = os.Remove(address[7:])
	})

	Describe("NewDefaultCOSIProvisionerClient", func() {
		It("should initialize a client with debug mode enabled", func(ctx SpecContext) {
			client, err := grpcfactory.NewDefaultCOSIProvisionerClient(ctx, address, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(client).NotTo(BeNil())
		})

		It("should fail if the address scheme is invalid", func(ctx SpecContext) {
			client, err := grpcfactory.NewDefaultCOSIProvisionerClient(ctx, "http://localhost", false)
			Expect(err).To(HaveOccurred())
			Expect(client).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("unsupported scheme"))
		})
	})

	Describe("NewCOSIProvisionerServer", func() {
		It("should initialize a server with valid arguments", func() {
			server, err := grpcfactory.NewCOSIProvisionerServer(address, mockIdentityServer, mockProvisionerServer, nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(server).NotTo(BeNil())
		})

		It("should return an error if identity server is nil", func() {
			server, err := grpcfactory.NewCOSIProvisionerServer(address, nil, mockProvisionerServer, nil)
			Expect(err).To(HaveOccurred())
			Expect(server).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Identity server cannot be nil"))
		})

		It("should return an error if provisioner server is nil", func() {
			server, err := grpcfactory.NewCOSIProvisionerServer(address, mockIdentityServer, nil, nil)
			Expect(err).To(HaveOccurred())
			Expect(server).To(BeNil())
			Expect(err.Error()).To(ContainSubstring("Provisioner server cannot be nil"))
		})
	})
})
