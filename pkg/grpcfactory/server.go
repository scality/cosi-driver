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

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
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

func (s *COSIProvisionerServer) Run(ctx context.Context, registry prometheus.Registerer) error {

	srvMetrics := grpcprom.NewServerMetrics(
		grpcprom.WithServerHandlingTimeHistogram(
			grpcprom.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
		),
	)
	if err := registry.Register(srvMetrics); err != nil {
		return fmt.Errorf("failed to register gRPC metrics: %w", err)
	}

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
		klog.ErrorS(err, "Failed to start listener")
		return fmt.Errorf("failed to start listener: %w", err)
	}

	defer func() {
		klog.Info("Closing listener...")
		listener.Close()
	}()

	s.listenOpts = append(s.listenOpts,
		grpc.ChainUnaryInterceptor(srvMetrics.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(srvMetrics.StreamServerInterceptor()),
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
