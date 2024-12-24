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
	"k8s.io/klog/v2"

	"github.com/scality/cosi-driver/pkg/grpcfactory"
	"github.com/scality/cosi-driver/pkg/metrics"
)

const (
	provisionerName       = "scality.com"
	defaultDriverAddress  = "unix:///var/lib/cosi/cosi.sock"
	defaultDriverPrefix   = "cosi"
	defaultMetricsPath    = "/metrics"
	defaultMetricsPrefix  = "scality_cosi_driver"
	defaultMetricsAddress = ":8080"
)

var (
	driverAddress        = flag.String("driver-address", defaultDriverAddress, "driver address for the socket file, default: unix:///var/lib/cosi/cosi.sock")
	driverPrefix         = flag.String("driver-prefix", defaultDriverPrefix, "prefix for COSI driver, e.g. <prefix>.scality.com, default cosi.scality.com")
	driverMetricsAddress = flag.String("driver-metrics-address", defaultMetricsAddress, "The address to expose Prometheus metrics, default: :8080")
	driverMetricsPath    = flag.String("driver-metrics-path", defaultMetricsPath, "path for the metrics endpoint, default: /metrics")
	driverMetricsPrefix  = flag.String("driver-custom-metrics-prefix", defaultMetricsPrefix, "prefix for the metrics, default: scality_cosi_driver_")
)

func init() {
	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Exitf("Failed to set logtostderr flag: %v", err)
	}
	flag.Parse()

	// check if driverMetricsPath starts with / if nor add it and chekc id it is path prood
	if !strings.HasPrefix(*driverMetricsPath, "/") {
		*driverMetricsPath = "/" + *driverMetricsPath
	}

	klog.InfoS("COSI driver startup configuration",
		"driverAddress", *driverAddress,
		"driverPrefix", *driverPrefix,
		"driverMetricsPath", *driverMetricsPath,
		"driverMetricsPrefix", *driverMetricsPrefix,
		"driverMetricsAddress", *driverMetricsAddress,
	)
}

func run(ctx context.Context) error {
	registry := prometheus.NewRegistry()
	driverName := *driverPrefix + "." + provisionerName
	metrics.InitializeMetrics(defaultMetricsPrefix, registry)

	metricsServer, err := metrics.StartMetricsServerWithRegistry(*driverMetricsAddress, registry, *driverMetricsPath)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	identityServer, bucketProvisioner, err := driver.CreateDriver(ctx, driverName)
	if err != nil {
		return fmt.Errorf("failed to initialize Scality driver: %w", err)
	}

	server, err := grpcfactory.NewDefaultCOSIProvisionerServer(*driverAddress, identityServer, bucketProvisioner)
	if err != nil {
		return fmt.Errorf("failed to start the provisioner server: %w", err)
	}

	err = server.Run(ctx, registry)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if shutdownErr := metricsServer.Shutdown(shutdownCtx); shutdownErr != nil {
		klog.ErrorS(shutdownErr, "Failed to gracefully shutdown metrics server")
	}

	return err
}
