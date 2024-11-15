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
	"context"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	cosi "sigs.k8s.io/container-object-storage-interface-spec"
)

type COSIProvisionerServer struct {
	address           string
	identityServer    cosi.IdentityServer
	provisionerServer cosi.ProvisionerServer
	listenOpts        []grpc.ServerOption
}

func (s *COSIProvisionerServer) Run(ctx context.Context) error {
	addr, err := url.Parse(s.address)
	if err != nil {
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
		klog.ErrorS(err, "Failed to start server")
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer listener.Close()
	server := grpc.NewServer(s.listenOpts...)
	cosi.RegisterIdentityServer(server, s.identityServer)
	cosi.RegisterProvisionerServer(server, s.provisionerServer)
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Serve(listener)
	}()
	select {
	case <-ctx.Done():
		server.GracefulStop()
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
