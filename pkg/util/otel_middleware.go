package util

import (
	"context"

	"github.com/aws/smithy-go/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/klog/v2"
)

// AttachOpenTelemetryMiddleware adds OpenTelemetry tracing to the middleware stack.
func AttachOpenTelemetryMiddleware(stack *middleware.Stack, serviceName string) error {
	tracingMiddleware := middleware.InitializeMiddlewareFunc("OpenTelemetryTracing", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler,
	) (out middleware.InitializeOutput, metadata middleware.Metadata, err error) {
		operationName := middleware.GetOperationName(ctx)

		// Start a new span.
		tracer := otel.Tracer("github.com/scality/cosi-driver")
		ctx, span := tracer.Start(ctx, serviceName+"/"+operationName, trace.WithSpanKind(trace.SpanKindClient))
		defer span.End()

		// Proceed with the AWS operation.
		out, metadata, err = next.HandleInitialize(trace.ContextWithSpan(ctx, span), in)

		if span != nil {
			// Add attributes to the span.
			span.SetAttributes(
				attribute.String("rpc.method", operationName),
				attribute.String("rpc.service", serviceName),
			)

			// Record errors and set span status.
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "AWS operation failed")
				klog.ErrorS(err, "AWS SDK operation failed", "operation", operationName)
			} else {
				span.SetStatus(codes.Ok, "AWS operation succeeded")
			}
		}

		return out, metadata, err
	})

	// Attach the middleware to the Initialize step.
	return stack.Initialize.Add(tracingMiddleware, middleware.After)
}
