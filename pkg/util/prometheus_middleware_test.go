package util_test

import (
	"context"
	"errors"

	"github.com/aws/smithy-go/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	u "github.com/scality/cosi-driver/pkg/util"
	"k8s.io/klog/v2"
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

	It("should safely execute the middleware chain", func(ctx SpecContext) {
		// Attach Prometheus middleware
		err := u.AttachPrometheusMiddleware(stack, requestDuration, requestsTotal)
		Expect(err).NotTo(HaveOccurred())

		// Add mock middleware to simulate behavior
		mockMiddleware := MockFinalizeMiddleware{
			HandleFunc: func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
				klog.InfoS("Mock middleware executed", "operation", middleware.GetOperationName(ctx))
				if next == nil {
					return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("next handler is nil")
				}
				return next.HandleFinalize(ctx, in)
			},
			IDValue: "MockMiddleware",
		}
		err = stack.Finalize.Add(mockMiddleware, middleware.Before)
		Expect(err).NotTo(HaveOccurred())

		// Ensure the chain is correctly constructed
		var handler middleware.FinalizeHandler = TerminalHandler{}
		for i := len(stack.Finalize.List()) - 1; i >= 0; i-- {
			middlewareID := stack.Finalize.List()[i]
			m, _ := stack.Finalize.Get(middlewareID)

			// Wrap the handler in a way that prevents infinite recursion
			previousHandler := handler
			handler = middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
				return m.HandleFinalize(ctx, in, previousHandler)
			})
		}

		// Execute the middleware chain
		_, _, err = handler.HandleFinalize(ctx, middleware.FinalizeInput{})
		Expect(err).NotTo(HaveOccurred())
	})

	// It("should record metrics with status 'success'", func() {
	// 	// Attach the Prometheus middleware
	// 	err := u.AttachPrometheusMiddleware(stack, requestDuration, requestsTotal)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Mock middleware simulating a successful operation
	// 	mockMiddleware := MockFinalizeMiddleware{
	// 		HandleFunc: func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
	// 			return next.HandleFinalize(ctx, in) // Call the next handler
	// 		},
	// 		IDValue: "MockSuccessMiddleware",
	// 	}
	// 	err = stack.Finalize.Add(mockMiddleware, middleware.Before) // Add mock middleware
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Build the middleware chain
	// 	var handler middleware.FinalizeHandler = TerminalHandler{}
	// 	for i := len(stack.Finalize.List()) - 1; i >= 0; i-- {
	// 		middlewareID := stack.Finalize.List()[i]
	// 		m, _ := stack.Finalize.Get(middlewareID)
	// 		handler = middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
	// 			return m.HandleFinalize(ctx, in, handler)
	// 		})
	// 	}

	// 	// Execute the middleware
	// 	_, _, err = handler.HandleFinalize(ctx, middleware.FinalizeInput{})
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Verify Prometheus metrics
	// 	Expect(testutil.CollectAndCount(requestDuration)).To(Equal(1))
	// 	Expect(testutil.ToFloat64(requestsTotal.WithLabelValues("TestOperation", "success"))).To(Equal(1.0))
	// })

	// It("should record metrics with status 'error'", func() {
	// 	// Attach the Prometheus middleware
	// 	err := u.AttachPrometheusMiddleware(stack, requestDuration, requestsTotal)
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Mock middleware simulating an error
	// 	mockMiddleware := MockFinalizeMiddleware{
	// 		HandleFunc: func(ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler) (middleware.FinalizeOutput, middleware.Metadata, error) {
	// 			return middleware.FinalizeOutput{}, middleware.Metadata{}, errors.New("mock error") // Simulate an error
	// 		},
	// 		IDValue: "MockErrorMiddleware",
	// 	}
	// 	err = stack.Finalize.Add(mockMiddleware, middleware.Before) // Add mock middleware
	// 	Expect(err).NotTo(HaveOccurred())

	// 	// Build the middleware chain
	// 	var handler middleware.FinalizeHandler = TerminalHandler{}
	// 	for i := len(stack.Finalize.List()) - 1; i >= 0; i-- {
	// 		middlewareID := stack.Finalize.List()[i]
	// 		m, _ := stack.Finalize.Get(middlewareID)
	// 		handler = middleware.FinalizeHandlerFunc(func(ctx context.Context, in middleware.FinalizeInput) (middleware.FinalizeOutput, middleware.Metadata, error) {
	// 			return m.HandleFinalize(ctx, in, handler)
	// 		})
	// 	}

	// 	// Execute the middleware
	// 	_, _, err = handler.HandleFinalize(ctx, middleware.FinalizeInput{})
	// 	Expect(err).To(HaveOccurred())

	// 	// Verify Prometheus metrics
	// 	Expect(testutil.CollectAndCount(requestDuration)).To(Equal(1))
	// 	Expect(testutil.ToFloat64(requestsTotal.WithLabelValues("TestOperation", "error"))).To(Equal(1.0))
	// })
})
