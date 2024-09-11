/*
Copyright 2024 Scality, Inc.
Licensed under the Apache License, Version 2.0 (the "License");
You may not use this file except in compliance with the License.
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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	cosiclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned"
	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

type provisionerServer struct {
	Provisioner   string
	KubeClientset *kubernetes.Clientset
	KubeConfig    *rest.Config
	CosiClientset cosiclientset.Interface
}

var _ cosiapi.ProvisionerServer = &provisionerServer{}

func InitProvisionerServer(driverName string) (cosiapi.ProvisionerServer, error) {
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	kubeClientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	cosiClientset, err := cosiclientset.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return &provisionerServer{
		Provisioner:   driverName,
		KubeClientset: kubeClientset,
		KubeConfig:    kubeConfig,
		CosiClientset: cosiClientset,
	}, nil
}

// DriverCreateBucket is an idempotent method for creating buckets
// It is expected to create the same bucket given a bucketName and protocol
// If the bucket already exists:
// - AND the parameters are the same, then it MUST return no error
// - AND the parameters are different, then it MUST return codes.AlreadyExists
//
// Return values
//
//	nil -                   Bucket successfully created
//	codes.AlreadyExists -   Bucket already exists. No more retries
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *provisionerServer) DriverCreateBucket(ctx context.Context,
	req *cosiapi.DriverCreateBucketRequest) (*cosiapi.DriverCreateBucketResponse, error) {

	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}

// DriverDeleteBucket is an idempotent method for deleting buckets
// It is expected to delete the same bucket given a bucketId
// If the bucket does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *provisionerServer) DriverDeleteBucket(ctx context.Context,
	req *cosiapi.DriverDeleteBucketRequest) (*cosiapi.DriverDeleteBucketResponse, error) {

	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}

// DriverCreateBucketAccess is an idempotent method for creating bucket access
// It is expected to create the same bucket access given a bucketId, name and protocol
//
// Return values
//
//	nil -                   Bucket access successfully created
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *provisionerServer) DriverGrantBucketAccess(ctx context.Context,
	req *cosiapi.DriverGrantBucketAccessRequest) (*cosiapi.DriverGrantBucketAccessResponse, error) {

	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}

// DriverDeleteBucketAccess is an idempotent method for deleting bucket access
// It is expected to delete the same bucket access given a bucketId and accountId
// If the bucket access does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket access successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *provisionerServer) DriverRevokeBucketAccess(ctx context.Context,
	req *cosiapi.DriverRevokeBucketAccessRequest) (*cosiapi.DriverRevokeBucketAccessResponse, error) {

	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}
