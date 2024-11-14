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
package grpcfactory

import (
	"google.golang.org/grpc"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

var (
	_ cosi.IdentityClient    = &COSIProvisionerClient{}
	_ cosi.ProvisionerClient = &COSIProvisionerClient{}
)

type COSIProvisionerClient struct {
	address string
	conn    *grpc.ClientConn
	cosi.IdentityClient
	cosi.ProvisionerClient
}
