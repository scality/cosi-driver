package metrics

import (
	"context"

	"github.com/aws/smithy-go/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

var AttachPrometheusMiddleware = attachPrometheusMiddlewareMetrics

// AttachPrometheusMiddleware attaches a Prometheus middleware for metrics tracking.
func attachPrometheusMiddlewareMetrics(stack *middleware.Stack, requestDuration *prometheus.HistogramVec, requestsTotal *prometheus.CounterVec) error {
	middlewareFunc := middleware.FinalizeMiddlewareFunc("PrometheusMetrics", func(
		ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
	) (out middleware.FinalizeOutput, metadata middleware.Metadata, err error) {
		operationName := middleware.GetOperationName(ctx)

		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(duration float64) {
			status := "success"
			if err != nil {
				status = "error"
			}

			requestDuration.WithLabelValues(operationName, status).Observe(duration)
			requestsTotal.WithLabelValues(operationName, status).Inc()
		}))
		defer timer.ObserveDuration()

		out, metadata, err = next.HandleFinalize(ctx, in)
		if err != nil {
			klog.ErrorS(err, "AWS SDK operation failed", "operation", operationName)
		}
		return out, metadata, err
	})

	// Add the middleware to the Finalize step
	return stack.Finalize.Add(middlewareFunc, middleware.After)
}
