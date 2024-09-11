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
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"

	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

type identityServer struct {
	provisioner string
}

var _ cosiapi.IdentityServer = &identityServer{}

func InitIdentityServer(driverName string) (cosiapi.IdentityServer, error) {
	if driverName == "" {
		return nil, fmt.Errorf("driver name must not be empty")
	}
	return &identityServer{
		provisioner: driverName,
	}, nil
}

func (id *identityServer) DriverGetInfo(ctx context.Context,
	req *cosiapi.DriverGetInfoRequest) (*cosiapi.DriverGetInfoResponse, error) {

	if id.provisioner == "" {
		klog.ErrorS(fmt.Errorf("provisioner name cannot be empty"), "invalid argument")
		return nil, status.Error(codes.InvalidArgument, "Provisioner name is empty")
	}

	return &cosiapi.DriverGetInfoResponse{
		Name: id.provisioner,
	}, nil
}
