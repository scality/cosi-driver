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

	"github.com/scality/cosi-driver/pkg/driver"
	"k8s.io/klog/v2"

	"github.com/scality/cosi-driver/pkg/provisioner"
)

const (
	provisionerName     = "scality.com"
	defaultDriverPrefix = "cosi"
)

var (
	driverAddress = flag.String("driver-address", "unix:///var/lib/cosi/cosi.sock", "driver address for the socket")
	driverPrefix  = flag.String("driver-prefix", "", "prefix for COSI driver, e.g. <prefix>.scality.com")
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
	driverName := *driverPrefix + "." + provisionerName

	identityServer, bucketProvisioner, err := driver.CreateDriver(ctx, driverName)
	if err != nil {
		return fmt.Errorf("failed to initialize Scality driver: %w", err)
	}

	server, err := provisioner.NewDefaultCOSIProvisionerServer(*driverAddress, identityServer, bucketProvisioner)
	if err != nil {
		return fmt.Errorf("failed to start the provisioner server: %w", err)
	}

	return server.Run(ctx)
}
