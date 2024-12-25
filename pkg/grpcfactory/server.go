package grpcfactory

import (
	"context"
	"fmt"
	"net"
	"net/url"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

// Helper function to initialize OpenTelemetry.
func initOpenTelemetry(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracing exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return tp, nil
}

// COSIProvisionerServer represents the gRPC server for the provisioner.
type COSIProvisionerServer struct {
	address           string
	identityServer    cosi.IdentityServer
	provisionerServer cosi.ProvisionerServer
	listenOpts        []grpc.ServerOption
}

// Run starts the gRPC server and handles incoming requests.
func (s *COSIProvisionerServer) Run(ctx context.Context, registry prometheus.Registerer) error {
	// Initialize OpenTelemetry.
	tp, err := initOpenTelemetry(ctx)
	if err != nil {
		klog.ErrorS(err, "Failed to initialize OpenTelemetry")
		return err
	}
	defer func() {
		if shutdownErr := tp.Shutdown(ctx); shutdownErr != nil {
			klog.ErrorS(shutdownErr, "Failed to shut down tracer provider")
		}
	}()

	// Set up Prometheus metrics.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	if err := registry.Register(srvMetrics); err != nil {
		klog.ErrorS(err, "Failed to register gRPC metrics")
		return fmt.Errorf("failed to register gRPC metrics: %w", err)
	}

	// Example: Function to extract exemplars from the tracing context.
	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		if span := trace.SpanContextFromContext(ctx); span.IsSampled() {
			return prometheus.Labels{"traceID": span.TraceID().String()}
		}
		return nil
	}

	// Parse the server address.
	addr, err := url.Parse(s.address)
	if err != nil {
		klog.ErrorS(err, "Invalid server address")
		return err
	}
	if addr.Scheme != "unix" {
		err := fmt.Errorf("unsupported scheme: expected 'unix', found '%s'", addr.Scheme)
		klog.ErrorS(err, "Invalid address scheme")
		return err
	}
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "unix", addr.Path)
	if err != nil {
		klog.ErrorS(err, "Failed to start listener")
		return fmt.Errorf("failed to start listener: %w", err)
	}
	defer func() {
		klog.Info("Closing listener...")
		if closeErr := listener.Close(); closeErr != nil {
			klog.ErrorS(closeErr, "Failed to close listener")
		}
	}()

	// Create the OpenTelemetry stats handler.
	otelHandler := otelgrpc.NewServerHandler()

	// Add interceptors in the correct order.
	s.listenOpts = append(s.listenOpts,
		grpc.StatsHandler(otelHandler), // Register the stats handler for OpenTelemetry.
		grpc.ChainUnaryInterceptor(
			srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)), // Metrics use tracing spans.
			func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
				traceID := trace.SpanContextFromContext(ctx).TraceID().String()
				klog.V(2).InfoS("Handling gRPC unary request", "method", info.FullMethod, "traceID", traceID)
				resp, err = handler(ctx, req)
				if err != nil {
					klog.ErrorS(err, "Error handling gRPC unary request", "method", info.FullMethod, "traceID", traceID)
				}
				return resp, err
			}, // Logging with klog.
		),
		grpc.ChainStreamInterceptor(
			srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext)), // Metrics use tracing spans.
			func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				traceID := trace.SpanContextFromContext(ss.Context()).TraceID().String()
				klog.V(2).InfoS("Handling gRPC stream request", "method", info.FullMethod, "traceID", traceID)
				err := handler(srv, ss)
				if err != nil {
					klog.ErrorS(err, "Error handling gRPC stream request", "method", info.FullMethod, "traceID", traceID)
				}
				return err
			}, // Logging with klog.
		),
	)

	server := grpc.NewServer(s.listenOpts...)
	cosi.RegisterIdentityServer(server, s.identityServer)
	cosi.RegisterProvisionerServer(server, s.provisionerServer)

	srvMetrics.InitializeMetrics(server)

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Serve(listener)
	}()
	select {
	case <-ctx.Done():
		klog.Info("Context canceled, stopping gRPC server...")
		server.GracefulStop()
		return ctx.Err()
	case err := <-errChan:
		klog.ErrorS(err, "gRPC server exited with error")
		return err
	}
}
