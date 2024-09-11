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

package driver

import (
	"context"

	"k8s.io/klog/v2"
	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

// CreateDriver initializes both the IdentityServer and ProvisionerServer for the COSI driver
func CreateDriver(ctx context.Context, driverName string) (cosiapi.IdentityServer, cosiapi.ProvisionerServer, error) {
	provisioner, err := InitProvisionerServer(driverName)
	if err != nil {
		klog.ErrorS(err, "Provisioner server initialization failed", "driverName", driverName)
		return nil, nil, err
	}

	identity, err := InitIdentityServer(driverName)
	if err != nil {
		klog.ErrorS(err, "Identity server initialization failed", "driverName", driverName)
		return nil, nil, err
	}

	return identity, provisioner, nil
}
