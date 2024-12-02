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
	"errors"
	"os"

	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	config "github.com/scality/cosi-driver/pkg/util/config"
	"github.com/scality/cosi-driver/pkg/util/iamclient"
	s3client "github.com/scality/cosi-driver/pkg/util/s3client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned"
	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

type ProvisionerServer struct {
	Provisioner     string
	Clientset       kubernetes.Interface
	KubeConfig      *rest.Config
	BucketClientset bucketclientset.Interface
}

var _ cosiapi.ProvisionerServer = &ProvisionerServer{}

// helper methods initialized as variables for testing
var InitializeClient = initializeObjectStorageClient
var FetchSecretInformation = fetchObjectStorageProviderSecretInfo
var FetchParameters = fetchS3Parameters

func InitProvisionerServer(provisioner string) (cosiapi.ProvisionerServer, error) {
	klog.V(3).InfoS("Initializing ProvisionerServer", "provisioner", provisioner)

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		klog.ErrorS(err, "Failed to get in-cluster config")
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create Kubernetes clientset")
		return nil, err
	}

	bucketClientset, err := bucketclientset.NewForConfig(kubeConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create BucketClientset")
		return nil, err
	}

	klog.V(3).InfoS("Successfully initialized ProvisionerServer", "provisioner", provisioner)
	return &ProvisionerServer{
		Provisioner:     provisioner,
		Clientset:       clientset,
		KubeConfig:      kubeConfig,
		BucketClientset: bucketClientset,
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
func (s *ProvisionerServer) DriverCreateBucket(ctx context.Context,
	req *cosiapi.DriverCreateBucketRequest) (*cosiapi.DriverCreateBucketResponse, error) {
	bucketName := req.GetName()
	parameters := req.GetParameters()
	clientType := "s3"

	klog.V(3).InfoS("Received DriverCreateBucket request", "bucketName", bucketName)
	klog.V(5).InfoS("Processing DriverCreateBucket", "bucketName", bucketName, "parameters", parameters)

	client, s3Params, err := InitializeClient(ctx, s.Clientset, parameters, clientType)
	if err != nil {
		klog.ErrorS(err, "Failed to initialize object storage provider S3 client", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider S3 client")
	}

	s3Client, ok := client.(*s3client.S3Client)
	if !ok {
		klog.ErrorS(nil, "Unsupported client type for bucket creation", "bucketName", bucketName)
		return nil, status.Error(codes.InvalidArgument, "unsupported client type for bucket creation")
	}

	err = s3Client.CreateBucket(ctx, bucketName, *s3Params)
	if err != nil {
		var bucketAlreadyExists *s3types.BucketAlreadyExists
		var bucketOwnedByYou *s3types.BucketAlreadyOwnedByYou

		if errors.As(err, &bucketAlreadyExists) {
			klog.V(3).InfoS("Bucket already exists", "bucketName", bucketName)
			return nil, status.Errorf(codes.AlreadyExists, "Bucket already exists: %s", bucketName)
		} else if errors.As(err, &bucketOwnedByYou) {
			klog.V(3).InfoS("A bucket with this name exists and is already owned by you: success", "bucketName", bucketName)
			return &cosiapi.DriverCreateBucketResponse{
				BucketId: bucketName,
			}, nil
		} else {
			var opErr *smithy.OperationError
			if errors.As(err, &opErr) {
				klog.V(4).InfoS("AWS operation error", "operation", opErr.OperationName, "message", opErr.Err.Error(), "bucketName", bucketName)
			}
			klog.ErrorS(err, "Failed to create bucket", "bucketName", bucketName)
			return nil, status.Error(codes.Internal, "Failed to create bucket")
		}
	}
	klog.V(3).InfoS("Successfully created bucket", "bucketName", bucketName)
	return &cosiapi.DriverCreateBucketResponse{
		BucketId: bucketName,
	}, nil
}

func initializeObjectStorageClient(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, clientType string) (interface{}, *config.StorageClientParameters, error) {
	klog.V(3).InfoS("Initializing object storage provider clients", "parameters", parameters)

	ospSecretName, namespace, err := FetchSecretInformation(parameters)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object storage provider secret info")
		return nil, nil, err
	}

	klog.V(4).InfoS("Fetching secret", "secretName", ospSecretName, "namespace", namespace)
	ospSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, ospSecretName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to get object store user secret", "secretName", ospSecretName)
		return nil, nil, status.Error(codes.Internal, "failed to get object store user secret")
	}

	storageClientParameters, err := FetchParameters(ospSecret.Data)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch S3 parameters from secret", "secretName", ospSecretName)
		return nil, nil, err
	}

	// clientType := parameters["clientType"]

	var client interface{}

	switch clientType {
	case "IAM":
		klog.V(3).InfoS("Initializing IAM client", "endpoint", storageClientParameters.Endpoint)
		client, err = iamclient.InitIAMClient(*storageClientParameters)
		if err != nil {
			klog.ErrorS(err, "Failed to create IAM client", "endpoint", storageClientParameters.Endpoint)
			return nil, nil, status.Error(codes.Internal, "failed to create IAM client")
		}
		klog.V(3).InfoS("Successfully initialized IAM client", "endpoint", storageClientParameters.Endpoint)
	case "S3":
		klog.V(3).InfoS("Initializing S3 client (default)", "endpoint", storageClientParameters.Endpoint)
		client, err = s3client.InitClient(*storageClientParameters)
		if err != nil {
			klog.ErrorS(err, "Failed to create S3 client", "endpoint", storageClientParameters.Endpoint)
			return nil, nil, status.Error(codes.Internal, "failed to create S3 client")
		}
		klog.V(3).InfoS("Successfully initialized S3 client", "endpoint", storageClientParameters.Endpoint)
	default:
		klog.ErrorS(nil, "Unsupported client type", "clientType", clientType)
		return nil, nil, status.Error(codes.InvalidArgument, "unsupported client type")
	}
	klog.V(3).InfoS("Successfully initialized S3 client", "endpoint", storageClientParameters.Endpoint)
	return client, storageClientParameters, nil
}

func fetchObjectStorageProviderSecretInfo(parameters map[string]string) (string, string, error) {
	klog.V(4).InfoS("Fetching object storage provider secret info", "parameters", parameters)

	secretName := parameters["COSI_DRIVER_SECRET_NAME"]
	namespace := os.Getenv("POD_NAMESPACE")
	if parameters["COSI_DRIVER_SECRET_NAMESPACE"] != "" {
		namespace = parameters["COSI_DRIVER_SECRET_NAMESPACE"]
	}
	if secretName == "" || namespace == "" {
		klog.ErrorS(nil, "Missing object storage provider secret name or namespace", "secretName", secretName, "namespace", namespace)
		return "", "", status.Error(codes.InvalidArgument, "Object storage provider secret name and namespace are required")
	}

	klog.V(4).InfoS("Object storage provider secret info fetched", "secretName", secretName, "namespace", namespace)
	return secretName, namespace, nil
}

func fetchS3Parameters(secretData map[string][]byte) (*config.StorageClientParameters, error) {
	klog.V(5).InfoS("Fetching S3 parameters from secret")

	accessKey := string(secretData["COSI_DRIVER_OSP_ACCESS_KEY_ID"])
	secretKey := string(secretData["COSI_DRIVER_OSP_SECRET_ACCESS_KEY"])
	endpoint := string(secretData["COSI_DRIVER_OSP_ENDPOINT"])
	region := string(secretData["COSI_DRIVER_OSP_REGION"])

	if endpoint == "" || accessKey == "" || secretKey == "" || region == "" {
		klog.ErrorS(nil, "Missing required S3 parameters", "accessKey", accessKey != "", "secretKey", secretKey != "", "endpoint", endpoint != "", "region", region != "")
		return nil, status.Error(codes.InvalidArgument, "endpoint, accessKeyID, secretKey and region are required")
	}

	var tlsCert []byte
	if cert, exists := secretData["COSI_DRIVER_OSP_TLS_CERT_SECRET_NAME"]; exists {
		tlsCert = cert
	} else {
		klog.V(5).InfoS("TLS certificate is not provided, proceeding without it")
	}

	return &config.StorageClientParameters{
		AccessKey: accessKey,
		SecretKey: secretKey,
		Endpoint:  endpoint,
		Region:    region,
		TLSCert:   tlsCert,
	}, nil
}

// DriverDeleteBucket is an idempotent method for deleting buckets
// It is expected to delete the same bucket given a bucketId
// If the bucket does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) DriverDeleteBucket(ctx context.Context,
	req *cosiapi.DriverDeleteBucketRequest) (*cosiapi.DriverDeleteBucketResponse, error) {
	klog.V(3).InfoS("Received DriverDeleteBucket request", "bucketId", req.GetBucketId())
	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}

// DriverCreateBucketAccess is an idempotent method for creating bucket access
// It is expected to create the same bucket access given a bucketId, name and protocol
//
// Return values
//
//	nil -                   Bucket access successfully created
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) DriverGrantBucketAccess(ctx context.Context,
	req *cosiapi.DriverGrantBucketAccessRequest) (*cosiapi.DriverGrantBucketAccessResponse, error) {
	bucketName := req.GetBucketId()
	userName := req.GetName()
	parameters := req.GetParameters()

	klog.V(3).InfoS("Received DriverGrantBucketAccess request", "bucketName", bucketName, "userName", userName)
	klog.V(4).InfoS("Processing DriverGrantBucketAccess", "parameters", parameters)
	klog.V(5).InfoS("Request DriverGrantBucketAccess", "req", req)

	ospSecretName, namespace, err := FetchSecretInformation(parameters)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object storage provider secret info")
		return nil, status.Error(codes.Internal, "failed to fetch object storage provider secret info")
	}
	klog.V(4).InfoS("Fetching secret", "secretName", ospSecretName, "namespace", namespace)
	ospSecret, err := s.Clientset.CoreV1().Secrets(namespace).Get(ctx, ospSecretName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to get object store user secret", "secretName", ospSecretName)
		return nil, status.Error(codes.Internal, "failed to get object store user secret")
	}

	accessKey := string(ospSecret.Data["COSI_DRIVER_OSP_ACCESS_KEY_ID"])
	secretKey := string(ospSecret.Data["COSI_DRIVER_OSP_SECRET_ACCESS_KEY"])
	endpoint := string(ospSecret.Data["COSI_DRIVER_OSP_ENDPOINT"])
	region := string(ospSecret.Data["COSI_DRIVER_OSP_REGION"])

	if endpoint == "" || accessKey == "" || secretKey == "" || region == "" {
		klog.ErrorS(nil, "Missing required IAM parameters", "accessKey", accessKey != "", "secretKey", secretKey != "", "endpoint", endpoint != "", "region", region != "")
		return nil, status.Error(codes.InvalidArgument, "endpoint, accessKeyID, secretKey and region are required")
	}

	// if err != nil {
	// 	klog.ErrorS(err, "Failed to fetch IAM parameters from secret", "secretName", ospSecretName)
	// 	return nil, status.Error(codes.Internal, "failed to fetch IAM parameters from secret")
	// }

	// var tlsCert []byte
	// if cert, exists := ospSecret.Data["COSI_DRIVER_OSP_TLS_CERT_SECRET_NAME"]; exists {
	// 	tlsCert = cert
	// } else {
	// 	klog.V(5).InfoS("TLS certificate is not provided, proceeding without it")
	// }

	// iamClientParams := config.StorageClientParameters{
	// 	AccessKey: accessKey,
	// 	SecretKey: secretKey,
	// 	Endpoint:  endpoint,
	// 	Region:    region,
	// 	TLSCert:   tlsCert,
	// }

	iamClient, err := iamclient.InitIAMClient(iamClientParams)
	if err != nil {
		klog.ErrorS(err, "Failed to create IAM client", "endpoint", iamClientParams.Endpoint)
		return nil, status.Error(codes.Internal, "failed to create IAM client")
	}

	userInfo, err := iamClient.CreateBucketAccess(ctx, bucketName, userName)
	if err != nil {
		klog.ErrorS(err, "Failed to create bucket access", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to create bucket access")
	}

	klog.V(3).InfoS("Successfully created bucket access", "bucketName", bucketName, "userName", userName)
	return &cosiapi.DriverGrantBucketAccessResponse{
		AccountId: userName,
		Credentials: map[string]*cosiapi.CredentialDetails{
			"s3": {
				Secrets: map[string]string{
					"accessKeyID":     *userInfo.AccessKey.AccessKeyId,
					"accessSecretKey": *userInfo.AccessKey.SecretAccessKey,
				},
			},
		},
	}, nil
}

// DriverDeleteBucketAccess is an idempotent method for deleting bucket access
// It is expected to delete the same bucket access given a bucketId and accountId
// If the bucket access does not exist, then it MUST return no error
//
// Return values
//
//	nil -                   Bucket access successfully deleted
//	non-nil err -           Internal error                                [requeue'd with exponential backoff]
func (s *ProvisionerServer) DriverRevokeBucketAccess(ctx context.Context,
	req *cosiapi.DriverRevokeBucketAccessRequest) (*cosiapi.DriverRevokeBucketAccessResponse, error) {
	klog.V(3).InfoS("Received DriverRevokeBucketAccess request", "bucketId", req.GetBucketId(), "accountId", req.GetAccountId())
	return nil, status.Error(codes.Unimplemented, "DriverCreateBucket: not implemented")
}
