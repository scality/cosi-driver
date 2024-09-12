package driver_test

import (
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi/pkg/driver"

	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

var _ = Describe("Identity Server DriverGetInfo", func() {
	Context("with valid provisioner names", func() {
		var (
			request *cosiapi.DriverGetInfoRequest
			server  cosiapi.IdentityServer
			err     error
		)

		// Helper function to initialize the server with the given provisioner and perform DriverGetInfo
		initAndGetInfo := func(provisionerName string) (*cosiapi.DriverGetInfoResponse, error) {
			server, err = driver.InitIdentityServer(provisionerName)
			Expect(err).ToNot(HaveOccurred())
			return server.DriverGetInfo(context.Background(), request)
		}

		BeforeEach(func() {
			request = &cosiapi.DriverGetInfoRequest{}
		})

		It("should return default driver name info", func() {
			provisionerName := "scality-cosi-driver"
			resp, err := initAndGetInfo(provisionerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Name).To(Equal(provisionerName))
		})

		It("should return a long driver name info", func() {
			provisionerName := "scality-cosi-driver" + strings.Repeat("x", 1000)
			resp, err := initAndGetInfo(provisionerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Name).To(Equal(provisionerName))
		})

		It("should return driver name info containing special characters", func() {
			provisionerName := "scality-cosi-driver-ß∂ƒ©"
			resp, err := initAndGetInfo(provisionerName)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Name).To(Equal(provisionerName))
		})
	})

	Context("with invalid provisioner names", func() {
		var (
			err error
		)

		It("should return an error for empty driver name", func() {
			_, err = driver.InitIdentityServer("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("driver name must not be empty"))
		})
	})

	// For now no use of request object in DriverInfo
	// Test to ensure intentional changes in the future
	Context("with nil request object", func() {
		var (
			server cosiapi.IdentityServer
			err    error
		)

		BeforeEach(func() {
			server, err = driver.InitIdentityServer("scality-cosi-driver")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle nil request gracefully", func() {
			resp, err := server.DriverGetInfo(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp).ToNot(BeNil())
			Expect(resp.Name).To(Equal("scality-cosi-driver"))
		})
	})
})
