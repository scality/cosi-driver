package util

import (
	"context"

	"github.com/aws/smithy-go/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

// AttachOpenTelemetryMiddleware attaches OpenTelemetry middleware for tracing AWS SDK operations.
func AttachOpenTelemetryMiddleware(stack *middleware.Stack, serviceName string, requestDuration *prometheus.HistogramVec, requestsTotal *prometheus.CounterVec) error {
	klog.Info("AttachOpenTelemetryMiddleware")
	// Initialize middleware: Start the span and propagate context.
	initializeMiddleware := middleware.InitializeMiddlewareFunc("OpenTelemetryTracingInitialize", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler,
	) (out middleware.InitializeOutput, metadata middleware.Metadata, err error) {
		operationName := middleware.GetOperationName(ctx)

		// Start a new span with the current context.
		tracer := otel.Tracer("github.com/scality/cosi-driver/s3client")
		ctx, span := tracer.Start(ctx, operationName, trace.WithSpanKind(trace.SpanKindClient))

		// Pass the updated context to the next middleware.
		return next.HandleInitialize(trace.ContextWithSpan(ctx, span), in)
	})

	// Finalize middleware: End the span and record metrics.
	finalizeMiddleware := middleware.FinalizeMiddlewareFunc("OpenTelemetryTracingFinalize", func(
		ctx context.Context, in middleware.FinalizeInput, next middleware.FinalizeHandler,
	) (out middleware.FinalizeOutput, metadata middleware.Metadata, err error) {
		klog.Info("OpenTelemetryTracingFinalize")
		// Retrieve the current span from the context.
		span := trace.SpanFromContext(ctx)

		operationName := middleware.GetOperationName(ctx)
		timer := prometheus.NewTimer(prometheus.ObserverFunc(func(duration float64) {
			status := "success"
			if err != nil {
				status = "error"
			}
			traceID := ""
			if span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}
			requestDuration.WithLabelValues(operationName, status, traceID).Observe(duration)
			requestsTotal.WithLabelValues(operationName, status, traceID).Inc()
		}))
		defer timer.ObserveDuration()

		// Proceed with the AWS operation.
		out, metadata, err = next.HandleFinalize(ctx, in)

		if span != nil {
			// Add attributes to the span.
			span.SetAttributes(
				attribute.String("rpc.method", operationName),
				attribute.String("rpc.service", serviceName),
			)

			// Extract request ID from metadata and add it to the span.
			requestID := getRequestID(metadata)
			if requestID != "" {
				span.SetAttributes(attribute.String("aws.request_id", requestID))
			}

			// Record errors and set span status.
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "AWS operation failed")
				klog.ErrorS(err, "AWS SDK operation failed", "operation", operationName, "request_id", requestID)
			} else {
				span.SetStatus(codes.Ok, "AWS operation succeeded")
			}

			// End the span.
			span.End()
		}

		return out, metadata, err
	})

	// Attach the middleware to the stack.
	if err := stack.Initialize.Add(initializeMiddleware, middleware.After); err != nil {
		return err
	}
	if err := stack.Finalize.Add(finalizeMiddleware, middleware.After); err != nil {
		return err
	}
	return nil
}

// getRequestID retrieves the AWS request ID from middleware metadata.
func getRequestID(metadata middleware.Metadata) string {
	// The AWS SDK uses specific keys to store request metadata. "x-amz-request-id" is common.
	if value := metadata.Get("x-amz-request-id"); value != nil {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}
	return ""
}
