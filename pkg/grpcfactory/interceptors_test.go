package grpcfactory_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	"google.golang.org/grpc"
)

var _ = Describe("gRPC Factory Interceptors", func() {
	var (
		ctx        context.Context
		method     string
		req, reply interface{}
		cc         *grpc.ClientConn
	)

	BeforeEach(func() {
		ctx = context.Background()
		method = "TestMethod"
		req = "test request"
		reply = "test reply"
		cc = &grpc.ClientConn{}
	})

	Context("apiLogger", func() {
		It("should log request and response successfully", func() {
			invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				// Simulate a successful invocation
				time.Sleep(50 * time.Millisecond)
				return nil
			}

			err := grpcfactory.ApiLogger(ctx, method, req, reply, cc, invoker)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle invocation error and log it", func() {
			invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				// Simulate an invocation error
				return errors.New("invocation failed")
			}

			err := grpcfactory.ApiLogger(ctx, method, req, reply, cc, invoker)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("invocation failed"))
		})
	})
})
