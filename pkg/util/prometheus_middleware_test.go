package util_test

import (
	"context"
	"errors"

	"github.com/aws/smithy-go/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	u "github.com/scality/cosi-driver/pkg/util"
)

// MockFinalizeMiddleware satisfies the FinalizeMiddleware interface
type MockFinalizeMiddleware struct {
	HandleFunc func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error)
	IDValue    string
}

func (m MockFinalizeMiddleware) HandleFinalize(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
	if next == nil {
		return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("next handler is nil")
	}

	out, metadata, err := m.HandleFunc(ctx, in, next)
	return out, metadata, err
}

// ID returns the unique identifier for the middleware
func (m MockFinalizeMiddleware) ID() string {
	return m.IDValue
}

// TerminalHandler represents the final handler in the middleware chain
type TerminalHandler struct{}

// HandleFinalize simulates the terminal handler in the chain
func (t TerminalHandler) HandleFinalize(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
	return middleware.FinalizeOutput{}, middleware.Metadata{}, nil
}

var _ = Describe("AttachPrometheusMiddleware", func() {
	var (
		stack           *middleware.Stack
		requestDuration *prometheus.HistogramVec
		requestsTotal   *prometheus.CounterVec
		// ctx             context.Context
	)

	BeforeEach(func() {
		// Initialize Prometheus metric vectors
		requestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "request_duration_seconds",
			Help: "Duration of requests",
		}, []string{"operation", "status"})

		requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total number of requests",
		}, []string{"operation", "status"})

		// Create a middleware stack
		stack = middleware.NewStack("testStack", nil)

		// Context with operation name
		// ctx = middleware.WithOperationName(context.Background(), "TestOperation")
	})

	It("should attach the middleware to the stack", func() {
		// Attach the Prometheus middleware
		err := u.AttachPrometheusMiddleware(stack, requestDuration, requestsTotal)
		Expect(err).NotTo(HaveOccurred())

		// Verify middleware is in the stack
		Expect(stack.Finalize.List()).To(HaveLen(1))
		Expect(stack.Finalize.List()[0]).To(Equal("PrometheusMetrics"))
	})
})
