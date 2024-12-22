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
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/klog/v2"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		klog.InfoS("Signal received", "type", sig)
		cancel()

		klog.InfoS("Initiating graceful shutdown, repeat signal to force shutdown")

		select {
		case sig = <-sigs:
			klog.ErrorS(nil, "Force shutdown due to repeated signal", "type", sig)
			os.Exit(1)
		case <-time.After(30 * time.Second):
			klog.ErrorS(nil, "Force shutdown due to timeout", "timeout", 30*time.Second)
			os.Exit(1)
		}
	}()

	// Call the run function (defined in cmd.go)
	if err := run(ctx); err != nil {
		klog.ErrorS(err, "Scality COSI driver encountered an error, shutting down")
		os.Exit(1)
	}
}
