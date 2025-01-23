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
	iamclient "github.com/scality/cosi-driver/pkg/clients/iam"
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	c "github.com/scality/cosi-driver/pkg/constants"
	"github.com/scality/cosi-driver/pkg/util"
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
var (
	InClusterConfig     = rest.InClusterConfig
	NewKubernetesClient = func(c *rest.Config) (kubernetes.Interface, error) {
		return kubernetes.NewForConfig(c)
	}
	NewBucketClient = func(c *rest.Config) (bucketclientset.Interface, error) {
		return bucketclientset.NewForConfig(c)
	}
)
var InitializeClient = initializeObjectStorageClient
var FetchSecretInformation = fetchObjectStorageProviderSecretInfo
var FetchParameters = fetchS3Parameters

func InitProvisionerServer(provisioner string) (cosiapi.ProvisionerServer, error) {
	if provisioner == "" {
		err := errors.New("provisioner name cannot be empty")
		klog.ErrorS(err, "Failed to initialize ProvisionerServer: empty provisioner name")
		return nil, err
	}
	klog.V(c.LvlEvent).InfoS("Initializing ProvisionerServer", "provisioner", provisioner)

	kubeConfig, err := InClusterConfig()
	if err != nil {
		klog.ErrorS(err, "Failed to get in-cluster config")
		return nil, err
	}

	clientset, err := NewKubernetesClient(kubeConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create Kubernetes clientset")
		return nil, err
	}

	bucketClientset, err := NewBucketClient(kubeConfig)
	if err != nil {
		klog.ErrorS(err, "Failed to create BucketClientset")
		return nil, err
	}

	klog.V(c.LvlEvent).InfoS("Successfully initialized ProvisionerServer", "provisioner", provisioner)
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
	klog.V(c.LvlTrace).InfoS("DriverCreateBucket request received", "request", req)
	bucketName := req.GetName()
	parameters := req.GetParameters()
	service := "S3"

	klog.V(c.LvlInfo).InfoS("Processing DriverCreateBucket request", "bucketName", bucketName)

	client, s3Params, err := InitializeClient(ctx, s.Clientset, parameters, service)
	if err != nil {
		klog.ErrorS(err, "Failed to initialize S3 client", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider S3 client")
	}

	s3Client, ok := client.(*s3client.S3Client)
	if !ok {
		klog.ErrorS(nil, "Unsupported client type for bucket creation", "bucketName", bucketName)
		return nil, status.Error(codes.InvalidArgument, "unsupported client type for bucket creation")
	}

	klog.V(c.LvlDebug).InfoS("Creating bucket", "bucketName", bucketName)
	err = s3Client.CreateBucket(ctx, bucketName, *s3Params)
	if err != nil {
		var bucketAlreadyExists *s3types.BucketAlreadyExists

		if errors.As(err, &bucketAlreadyExists) {
			klog.V(c.LvlInfo).InfoS("Bucket already exists", "bucketName", bucketName)
			return nil, status.Errorf(codes.AlreadyExists, "Bucket already exists: %s", bucketName)
		} else {
			klog.ErrorS(err, "Failed to create bucket", "bucketName", bucketName)
			return nil, status.Error(codes.Internal, "Failed to create bucket")
		}
	}
	klog.V(c.LvlInfo).InfoS("Successfully created bucket", "bucketName", bucketName)
	return &cosiapi.DriverCreateBucketResponse{
		BucketId: bucketName,
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
	klog.V(c.LvlTrace).InfoS("DriverDeleteBucket request received", "request", req)

	bucketName := req.GetBucketId()

	klog.V(c.LvlInfo).InfoS("Processing DriverDeleteBucket request", "bucketName", bucketName)
	bucket, err := s.BucketClientset.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to fetch bucket object", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to get bucket object from kubernetes")
	}
	klog.V(c.LvlTrace).InfoS("Successfully fetched Bucket object", "bucketName", bucket.Name, "parameters", bucket.Spec.Parameters)

	client, _, err := InitializeClient(ctx, s.Clientset, bucket.Spec.Parameters, "S3")
	if err != nil {
		klog.ErrorS(err, "Failed to initialize S3 client for bucket deletion", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider S3 client")
	}

	s3Client, ok := client.(*s3client.S3Client)
	if !ok {
		klog.ErrorS(nil, "Unsupported client type for bucket deletion", "bucketName", bucketName)
		return nil, status.Error(codes.InvalidArgument, "unsupported client type for bucket deletion")
	}

	err = s3Client.DeleteBucket(ctx, bucketName)
	if err != nil {
		klog.ErrorS(err, "Failed to delete bucket", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to delete bucket")
	}

	klog.V(c.LvlInfo).InfoS("Successfully deleted bucket", "bucketName", bucketName)
	return &cosiapi.DriverDeleteBucketResponse{}, nil
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
	klog.V(c.LvlTrace).InfoS("DriverGrantBucketAccess request received", "request", req)

	bucketName := req.GetBucketId()
	userName := req.GetName()
	parameters := req.GetParameters()

	klog.V(c.LvlInfo).InfoS("Processing DriverGrantBucketAccess request", "bucketName", bucketName, "userName", userName)

	client, iamParams, err := InitializeClient(ctx, s.Clientset, parameters, "IAM")

	if err != nil {
		klog.ErrorS(err, "Failed to initialize IAM client", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider IAM client")
	}

	iamClient, ok := client.(*iamclient.IAMClient)
	if !ok {
		klog.ErrorS(nil, "Unsupported client type for bucket access", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider IAM client")
	}

	klog.V(c.LvlInfo).InfoS("Granting bucket access", "bucketName", bucketName, "userName", userName)
	userInfo, err := iamClient.CreateBucketAccess(ctx, userName, bucketName)
	if err != nil {
		klog.ErrorS(err, "Failed to create bucket access", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to create bucket access")
	}

	klog.V(c.LvlInfo).InfoS("Successfully granted bucket access", "bucketName", bucketName, "userName", userName)
	return &cosiapi.DriverGrantBucketAccessResponse{
		AccountId: userName,
		Credentials: map[string]*cosiapi.CredentialDetails{
			"s3": {
				Secrets: map[string]string{
					"accessKeyID":     *userInfo.AccessKey.AccessKeyId,
					"accessSecretKey": *userInfo.AccessKey.SecretAccessKey,
					"endpoint":        iamParams.Endpoint,
					"region":          iamParams.Region,
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
	klog.V(c.LvlTrace).InfoS("DriverRevokeBucketAccess request received", "request", req)

	bucketName := req.GetBucketId()
	userName := req.GetAccountId()

	klog.V(c.LvlInfo).InfoS("Processing DriverRevokeBucketAccess request", "bucketName", bucketName, "userName", userName)

	// Fetch the bucket to retrieve parameters
	bucket, err := s.BucketClientset.ObjectstorageV1alpha1().Buckets().Get(ctx, bucketName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to fetch bucket object", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "failed to get bucket object from kubernetes")
	}
	klog.V(c.LvlTrace).InfoS("Successfully fetched Bucket object", "bucketName", bucket.Name, "parameters", bucket.Spec.Parameters)

	client, _, err := InitializeClient(ctx, s.Clientset, bucket.Spec.Parameters, "IAM")
	if err != nil {
		klog.ErrorS(err, "Failed to initialize IAM client", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to initialize object storage provider IAM client")
	}

	iamClient, ok := client.(*iamclient.IAMClient)
	if !ok {
		klog.ErrorS(nil, "Unsupported client type for revoking bucket access", "bucketName", bucketName)
		return nil, status.Error(codes.Internal, "unsupported client type for IAM operations")
	}

	klog.V(c.LvlInfo).InfoS("Revoking bucket access", "bucketName", bucketName, "userName", userName)
	err = iamClient.RevokeBucketAccess(ctx, userName, bucketName)
	if err != nil {
		klog.ErrorS(err, "Failed to revoke bucket access", "bucketName", bucketName, "userName", userName)
		return nil, status.Error(codes.Internal, "failed to revoke bucket access")
	}

	klog.V(c.LvlInfo).InfoS("Successfully revoked bucket access", "bucketName", bucketName, "userName", userName)
	return &cosiapi.DriverRevokeBucketAccessResponse{}, nil
}

func initializeObjectStorageClient(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
	klog.V(c.LvlDebug).InfoS("Initializing object storage provider client", "service", service)

	ospSecretName, namespace, err := FetchSecretInformation(parameters)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object storage provider secret info")
		return nil, nil, err
	}

	klog.V(c.LvlDebug).InfoS("Fetching secret data", "secretName", ospSecretName, "namespace", namespace)
	ospSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, ospSecretName, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "Failed to get object store user secret", "secretName", ospSecretName, "namespace", namespace)
		return nil, nil, status.Error(codes.Internal, "failed to get object store user secret")
	}
	klog.V(c.LvlDebug).InfoS("Successfully fetched object storage provider secret", "secretName", ospSecretName, "namespace", namespace)

	storageClientParameters, err := FetchParameters(ospSecret.Data)
	if err != nil {
		klog.ErrorS(err, "Failed to fetch object storage provider parameters from secret", "secretName", ospSecretName)
		return nil, nil, err
	}

	var client interface{}
	switch service {
	case "S3":
		client, err = s3client.InitS3Client(ctx, *storageClientParameters)
		if err != nil {
			klog.ErrorS(err, "Failed to initialize S3 client", "endpoint", storageClientParameters.Endpoint)
			return nil, nil, status.Error(codes.Internal, "failed to initialize S3 client")
		}
		klog.V(c.LvlDebug).InfoS("Successfully initialized S3 client", "endpoint", storageClientParameters.Endpoint)
	case "IAM":
		client, err = iamclient.InitIAMClient(ctx, *storageClientParameters)
		if err != nil {
			klog.ErrorS(err, "Failed to initialize IAM client", "endpoint", storageClientParameters.IAMEndpoint)
			return nil, nil, status.Error(codes.Internal, "failed to initialize IAM client")
		}
		klog.V(c.LvlDebug).InfoS("Successfully initialized IAM client", "endpoint", storageClientParameters.IAMEndpoint)
	default:
		klog.ErrorS(nil, "Unsupported object storage provider service", "service", service)
		return nil, nil, status.Error(codes.Internal, "unsupported object storage provider service")
	}
	return client, storageClientParameters, nil
}

func fetchObjectStorageProviderSecretInfo(parameters map[string]string) (string, string, error) {
	klog.V(c.LvlDebug).InfoS("Validating object storage provider secret parameters", "parameters", parameters)

	secretName := parameters["objectStorageSecretName"]
	namespace := os.Getenv("POD_NAMESPACE")
	if parameters["objectStorageSecretNamespace"] != "" {
		namespace = parameters["objectStorageSecretNamespace"]
	}
	if secretName == "" || namespace == "" {
		klog.ErrorS(nil, "Missing object storage provider secret name or namespace", "secretName", secretName, "namespace", namespace)
		return "", "", status.Error(codes.InvalidArgument, "Object storage provider secret name and namespace are required")
	}

	klog.V(c.LvlDebug).InfoS("Successfully validated object storage provider secret parameters", "secretName", secretName, "namespace", namespace)
	return secretName, namespace, nil
}

func fetchS3Parameters(secretData map[string][]byte) (*util.StorageClientParameters, error) {
	klog.V(c.LvlTrace).InfoS("Extracting object storage parameters from secret")

	params := util.NewStorageClientParameters()

	params.AccessKeyID = string(secretData["accessKeyId"])
	params.SecretAccessKey = string(secretData["secretAccessKey"])
	params.Endpoint = string(secretData["endpoint"])
	params.Region = string(secretData["region"])

	if cert, exists := secretData["tlsCert"]; exists {
		params.TLSCert = cert
	} else {
		klog.V(c.LvlTrace).InfoS("TLS certificate not provided, proceeding without it")
	}

	if err := params.Validate(); err != nil {
		klog.ErrorS(err, "Invalid object storage parameters")
		return nil, err
	}

	params.IAMEndpoint = params.Endpoint
	if value, exists := secretData["iamEndpoint"]; exists && len(value) > 0 {
		params.IAMEndpoint = string(value)
		klog.V(c.LvlTrace).InfoS("IAM endpoint specified", "iamEndpoint", params.IAMEndpoint)
	}
	klog.V(c.LvlTrace).InfoS("Successfully validated object storage parameters", "endpoint", params.Endpoint, "region", params.Region)
	return params, nil
}
