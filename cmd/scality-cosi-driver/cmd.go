/*
Copyright 2024 Scality, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scality/cosi-driver/pkg/driver"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	"github.com/scality/cosi-driver/pkg/metrics"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"k8s.io/klog/v2"
)

const (
	provisionerName        = "scality.com"
	defaultDriverAddress   = "unix:///var/lib/cosi/cosi.sock"
	defaultDriverPrefix    = "cosi"
	defaultMetricsPath     = "/metrics"
	defaultMetricsPrefix   = "scality_cosi_driver"
	defaultMetricsAddress  = ":8080"
	defaultOtelStdout      = false
	defaultOtelEndpoint    = "localhost:4318"
	defaultOtelServiceName = "cosi.scality.com"
)

var (
	driverAddress         = flag.String("driver-address", defaultDriverAddress, "driver address for the socket file, default: unix:///var/lib/cosi/cosi.sock")
	driverPrefix          = flag.String("driver-prefix", defaultDriverPrefix, "prefix for COSI driver, e.g. <prefix>.scality.com, default: cosi")
	driverMetricsAddress  = flag.String("driver-metrics-address", defaultMetricsAddress, "The address (hostname:port) to expose Prometheus metrics, default: 0.0.0.0:8080")
	driverMetricsPath     = flag.String("driver-metrics-path", defaultMetricsPath, "path for the metrics endpoint, default: /metrics")
	driverMetricsPrefix   = flag.String("driver-custom-metrics-prefix", defaultMetricsPrefix, "prefix for the metrics, default: scality_cosi_driver")
	driverOtelEndpoint    = flag.String("driver-otel-endpoint", defaultOtelEndpoint, "OpenTelemetry endpoint to export traces, default: localhost:4318")
	driverOtelStdout      = flag.Bool("driver-otel-stdout", defaultOtelStdout, "Enable OpenTelemetry trace export to stdout, disables endpoint if enabled, default: false")
	driverOtelServiceName = flag.String("driver-otel-service-name", defaultOtelServiceName, "Service name for OpenTelemetry traces, default: cosi.scality.com")
)

func init() {
	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Exitf("Failed to set logtostderr flag: %v", err)
	}
	flag.Parse()

	// Ensure driverMetricsPath is properly formatted.
	if !strings.HasPrefix(*driverMetricsPath, "/") {
		*driverMetricsPath = "/" + *driverMetricsPath
	}
	if *driverOtelStdout && *driverOtelEndpoint != "" {
		klog.Warning("Both --driver-otel-stdout and --driver-otel-endpoint are set. Defaulting to stdout tracing.")
	}

	klog.InfoS("COSI driver startup configuration",
		"driverAddress", *driverAddress,
		"driverPrefix", *driverPrefix,
		"driverMetricsPath", *driverMetricsPath,
		"driverMetricsPrefix", *driverMetricsPrefix,
		"driverMetricsAddress", *driverMetricsAddress,
		"driverOtelEndpoint", *driverOtelEndpoint,
		"driverOtelStdout", *driverOtelStdout,
	)
}

// initOpenTelemetry initializes OpenTelemetry tracing.
func initOpenTelemetry(ctx context.Context) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if *driverOtelStdout {
		// Configure stdout exporter
		exporter, err = stdout.New(stdout.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("failed to initialize stdout exporter: %w", err)
		}
		klog.Info("OpenTelemetry tracing enabled with stdout exporter")
	} else {
		// Configure OTLP exporter
		exporter, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(*driverOtelEndpoint), otlptracehttp.WithInsecure())
		if err != nil {
			return nil, fmt.Errorf("failed to initialize OTLP exporter: %w", err)
		}
		klog.InfoS("OpenTelemetry tracing enabled with OTLP exporter", "endpoint", *driverOtelEndpoint)
	}
	// Set up the tracer provider with the selected exporter
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes("", attribute.String("service.name", *driverOtelServiceName))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Set error handler for OpenTelemetry
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		klog.ErrorS(err, "OpenTelemetry error")
	}))

	return tp, nil
}

func run(ctx context.Context) error {
	// Initialize metrics
	registry := prometheus.NewRegistry()
	metrics.InitializeMetrics(defaultMetricsPrefix, registry)

	metricsServer, err := metrics.StartMetricsServerWithRegistry(*driverMetricsAddress, registry, *driverMetricsPath)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := metricsServer.Shutdown(shutdownCtx); shutdownErr != nil {
			klog.ErrorS(shutdownErr, "Failed to gracefully shutdown metrics server")
		}
	}()

	// Initialize OpenTelemetry
	tp, err := initOpenTelemetry(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize OpenTelemetry: %w", err)
	}
	if tp != nil {
		defer func() {
			klog.Info("Shutting down OpenTelemetry tracer provider")
			if shutdownErr := tp.Shutdown(ctx); shutdownErr != nil {
				klog.ErrorS(shutdownErr, "Failed to shut down OpenTelemetry tracer provider")
			}
		}()
	}

	driverName := *driverPrefix + "." + provisionerName
	identityServer, bucketProvisioner, err := driver.CreateDriver(ctx, driverName)
	if err != nil {
		return fmt.Errorf("failed to initialize Scality driver: %w", err)
	}

	server, err := grpcfactory.NewDefaultCOSIProvisionerServer(*driverAddress, identityServer, bucketProvisioner)
	if err != nil {
		return fmt.Errorf("failed to start the provisioner server: %w", err)
	}

	err = server.Run(ctx, registry)
	shutdownCtx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	if shutdownErr := metricsServer.Shutdown(shutdownCtx); shutdownErr != nil {
		klog.ErrorS(shutdownErr, "Failed to gracefully shutdown metrics server")
	}

	return err
}
