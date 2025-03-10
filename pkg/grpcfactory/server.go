package grpcfactory

import (
	"context"
	"fmt"
	"net"
	"net/url"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

// COSIProvisionerServer represents the gRPC server for the provisioner.
type COSIProvisionerServer struct {
	address           string
	identityServer    cosi.IdentityServer
	provisionerServer cosi.ProvisionerServer
	listenOpts        []grpc.ServerOption
}

// Run starts the gRPC server and handles incoming requests.
func (s *COSIProvisionerServer) Run(ctx context.Context, registry prometheus.Registerer) error {
	// Set up Prometheus metrics with handling time histograms.
	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.01, 0.1, 0.5, 1, 2, 5}),
		),
	)

	// Function to extract exemplars from the tracing context.
	exemplarFromContext := func(ctx context.Context) prometheus.Labels {
		span := trace.SpanContextFromContext(ctx)
		if span.IsValid() && span.IsSampled() {
			traceID := span.TraceID().String()
			klog.V(5).InfoS("Adding exemplar with traceID", "traceID", traceID)
			return prometheus.Labels{"traceID": traceID}
		}
		klog.V(5).Info("Traces are not enabled or traceID is nil, no exemplar added")
		return nil
	}

	// Register metrics with the provided registry.
	if err := registry.Register(srvMetrics); err != nil {
		klog.ErrorS(err, "Failed to register gRPC metrics")
		return fmt.Errorf("failed to register gRPC metrics: %w", err)
	}

	// Parse and validate the server address.
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

	// Start the server listener.
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

	// Create the OpenTelemetry stats handler for instrumentation.
	otelHandler := otelgrpc.NewServerHandler()

	// Add gRPC server options including OpenTelemetry and Prometheus interceptors.
	// Add gRPC server options including OpenTelemetry and Prometheus interceptors.
	s.listenOpts = append(s.listenOpts,
		grpc.StatsHandler(otelHandler), // Register the stats handler for OpenTelemetry first.
		grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext))),
		grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor(grpcprom.WithExemplarFromContext(exemplarFromContext))),
	)

	// Initialize the gRPC server.
	server := grpc.NewServer(s.listenOpts...)
	cosi.RegisterIdentityServer(server, s.identityServer)
	cosi.RegisterProvisionerServer(server, s.provisionerServer)

	// Initialize metrics collection for the server.
	srvMetrics.InitializeMetrics(server)

	// Run the gRPC server and listen for incoming connections.
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
