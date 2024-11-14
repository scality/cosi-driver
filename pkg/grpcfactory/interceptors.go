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
	"encoding/json"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog/v2"
)

func ApiLogger(
	ctx context.Context,
	method string,
	req, reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	// Log the request
	if jsonReq, err := json.MarshalIndent(req, "", " "); err == nil {
		klog.InfoS("Request", "api", method, "req", string(jsonReq))
	} else {
		klog.ErrorS(err, "Failed to marshal request", "api", method)
	}

	start := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	elapsed := time.Since(start)

	// Log the response or error
	if err != nil {
		klog.ErrorS(err, "API call failed", "api", method, "elapsed", elapsed)
	} else if jsonResp, err := json.MarshalIndent(reply, "", " "); err == nil {
		klog.InfoS("Response", "api", method, "elapsed", elapsed, "resp", string(jsonResp))
	} else {
		klog.ErrorS(err, "Failed to marshal response", "api", method)
	}

	return err
}
