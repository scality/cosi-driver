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
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/scality/cosi-driver/pkg/driver"
	"k8s.io/klog/v2"

	c "github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/grpcfactory"
	"github.com/scality/cosi-driver/pkg/metrics"
)

const (
	provisionerName     = "scality.com"
	defaultDriverPrefix = "cosi"
)

var (
	driverAddress  = flag.String("driver-address", "unix:///var/lib/cosi/cosi.sock", "driver address for the socket")
	driverPrefix   = flag.String("driver-prefix", "", "prefix for COSI driver, e.g. <prefix>.scality.com")
	metricsAddress = flag.String("metrics-address", c.MetricsAddress, "The address to expose Prometheus metrics.")
)

func init() {
	klog.InitFlags(nil)
	if err := flag.Set("logtostderr", "true"); err != nil {
		klog.Exitf("Failed to set logtostderr flag: %v", err)
	}
	flag.Parse()

	if *driverPrefix == "" {
		*driverPrefix = defaultDriverPrefix
		klog.Warning("No driver prefix provided, using default prefix")
	}

	klog.InfoS("COSI driver startup configuration", "driverAddress", *driverAddress, "driverPrefix", *driverPrefix)
}

func run(ctx context.Context) error {
	registry := prometheus.NewRegistry()

	if err := registry.Register(metrics.RequestsTotal); err != nil {
		return fmt.Errorf("failed to register custom metrics: %w", err)
	}

	// Start the Prometheus metrics server with the shared registry
	metricsServer, err := metrics.StartMetricsServerWithRegistry(*metricsAddress, registry)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
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
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if shutdownErr := metricsServer.Shutdown(shutdownCtx); shutdownErr != nil {
		klog.ErrorS(shutdownErr, "Failed to gracefully shutdown metrics server")
	}

	return err
}
