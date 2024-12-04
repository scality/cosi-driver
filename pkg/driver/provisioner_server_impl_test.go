package driver_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	"github.com/scality/cosi-driver/pkg/driver"
	"github.com/scality/cosi-driver/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

type MockS3Client struct {
	CreateBucketFunc func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error)
}

func (m *MockS3Client) CreateBucket(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
	if m.CreateBucketFunc != nil {
		return m.CreateBucketFunc(ctx, input, opts...)
	}
	return &s3.CreateBucketOutput{}, nil
}

type MockIAMClient struct {
	CreateBucketAccessFunc func(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error)
}

// Mock CreateBucketAccess
func (m *MockIAMClient) CreateBucketAccess(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error) {
	if m.CreateBucketAccessFunc != nil {
		return m.CreateBucketAccessFunc(ctx, userName, bucketName)
	}
	return &iam.CreateAccessKeyOutput{
		AccessKey: &iamtypes.AccessKey{
			AccessKeyId:     aws.String("mock-access-key-id"),
			SecretAccessKey: aws.String("mock-secret-access-key"),
		},
	}, nil
}

var _ = Describe("ProvisionerServer DriverCreateBucket", Ordered, func() {
	var (
		mockS3                   *MockS3Client
		provisioner              *driver.ProvisionerServer
		ctx                      context.Context
		clientset                *fake.Clientset
		bucketName               string
		s3Params                 util.StorageClientParameters
		request                  *cosiapi.DriverCreateBucketRequest
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)
	)

	BeforeEach(func() {
		ctx = context.TODO()
		mockS3 = &MockS3Client{}
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Provisioner: "test-provisioner",
			Clientset:   clientset,
		}
		bucketName = "test-bucket"
		s3Params = util.StorageClientParameters{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Endpoint:        "https://test-endpoint",
			Region:          "us-west-2",
		}
		request = &cosiapi.DriverCreateBucketRequest{Name: bucketName}

		// Store the original function to restore it later
		originalInitializeClient = driver.InitializeClient

		// Mock InitializeClient with the correct signature
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			if service == "S3" {
				return &s3client.S3Client{S3Service: mockS3}, &s3Params, nil
			}
			return nil, nil, fmt.Errorf("unsupported service: %s", service)
		}
	})

	AfterEach(func() {
		// Restore the original InitializeClient function
		driver.InitializeClient = originalInitializeClient
	})

	It("should successfully create a new bucket", func() {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			Expect(input.Bucket).To(Equal(&bucketName))
			return &s3.CreateBucketOutput{}, nil
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.BucketId).To(Equal(bucketName))
	})

	It("should return AlreadyExists error if bucket already exists with different parameters", func() {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &types.BucketAlreadyExists{}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.AlreadyExists))
		Expect(err.Error()).To(ContainSubstring("Bucket already exists: test-bucket"))
	})

	It("should return success if bucket with same parameters already exists", func() {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &types.BucketAlreadyOwnedByYou{}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.BucketId).To(Equal(bucketName))
	})

	It("should return Internal error for other S3 client errors", func() {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, errors.New("SomeOtherError: Something went wrong")
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("Failed to create bucket"))
	})

	It("should return Internal error when InitializeClient fails", func() {
		// Mock InitializeClient to return an error
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock initialization error")
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider S3 client"))
	})

	It("should return InvalidArgument error when client type is unsupported", func() {
		// Mock InitializeClient to return an unsupported client type
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &struct{}{}, &s3Params, nil // Returning a struct instead of *s3client.S3Client
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("unsupported client type for bucket creation"))
	})
})

var _ = Describe("ProvisionerServer Unimplemented Methods", Ordered, func() {
	var (
		provisioner *driver.ProvisionerServer
		ctx         context.Context
		clientset   *fake.Clientset
		bucketName  string
		accountID   string
	)

	BeforeEach(func() {
		ctx = context.TODO()
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Provisioner: "test-provisioner",
			Clientset:   clientset,
		}
		bucketName = "test-bucket"
		accountID = "test-account-id"
	})

	It("DriverDeleteBucket should return Unimplemented error", func() {
		request := &cosiapi.DriverDeleteBucketRequest{BucketId: bucketName}
		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Unimplemented))
		Expect(err.Error()).To(ContainSubstring("DriverCreateBucket: not implemented"))
	})

	It("DriverRevokeBucketAccess should return Unimplemented error", func() {
		request := &cosiapi.DriverRevokeBucketAccessRequest{AccountId: accountID}
		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Unimplemented))
		Expect(err.Error()).To(ContainSubstring("DriverCreateBucket: not implemented"))
	})
})

var _ = Describe("FetchSecretInformation", Ordered, func() {
	var (
		parameters map[string]string
		secretName string
		namespace  string
	)

	BeforeEach(func() {
		parameters = make(map[string]string)
		secretName = "test-secret-name"
		namespace = "test-namespace"

		os.Unsetenv("POD_NAMESPACE")
	})

	AfterEach(func() {
		os.Unsetenv("POD_NAMESPACE")
	})

	It("should fetch secret name and namespace from parameters when all are provided", func() {
		parameters["objectStorageSecretName"] = secretName
		parameters["objectStorageSecretNamespace"] = namespace

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(fetchedSecretName).To(Equal(secretName))
		Expect(fetchedNamespace).To(Equal(namespace))
	})

	It("should use POD_NAMESPACE environment variable when namespace is not in parameters", func() {
		parameters["objectStorageSecretName"] = secretName
		os.Setenv("POD_NAMESPACE", namespace)

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(fetchedSecretName).To(Equal(secretName))
		Expect(fetchedNamespace).To(Equal(namespace))
	})

	It("should return error when secret name is missing", func() {
		parameters["objectStorageSecretNamespace"] = namespace

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(HaveOccurred())
		Expect(fetchedSecretName).To(BeEmpty())
		Expect(fetchedNamespace).To(BeEmpty())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should return error when namespace is missing and POD_NAMESPACE is not set", func() {
		parameters["objectStorageSecretName"] = secretName

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(HaveOccurred())
		Expect(fetchedSecretName).To(BeEmpty())
		Expect(fetchedNamespace).To(BeEmpty())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should prioritize namespace from parameters over POD_NAMESPACE environment variable", func() {
		parameters["objectStorageSecretName"] = secretName
		parameters["objectStorageSecretNamespace"] = namespace
		os.Setenv("POD_NAMESPACE", "env-namespace")

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(fetchedSecretName).To(Equal(secretName))
		Expect(fetchedNamespace).To(Equal(namespace))
	})

	It("should return error when both secret name and namespace are missing", func() {
		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(HaveOccurred())
		Expect(fetchedSecretName).To(BeEmpty())
		Expect(fetchedNamespace).To(BeEmpty())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})
})

var _ = Describe("initializeObjectStorageClient", Ordered, func() {
	var (
		ctx        context.Context
		clientset  *fake.Clientset
		parameters map[string]string
		secret     *corev1.Secret
	)

	BeforeEach(func() {
		ctx = context.TODO()
		clientset = fake.NewSimpleClientset()
		parameters = map[string]string{
			"objectStorageSecretName":      "test-secret",
			"objectStorageSecretNamespace": "test-namespace",
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"accessKeyId":     []byte("test-access-key"),
				"secretAccessKey": []byte("test-secret-key"),
				"endpoint":        []byte("https://test-endpoint"),
				"region":          []byte("us-west-2"),
			},
		}
	})

	It("should successfully initialize S3 client and parameters", func() {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(BeNil())
		Expect(s3Client).NotTo(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.AccessKeyID).To(Equal("test-access-key"))
		Expect(s3Params.SecretAccessKey).To(Equal("test-secret-key"))
		Expect(s3Params.Endpoint).To(Equal("https://test-endpoint"))
		Expect(s3Params.Region).To(Equal("us-west-2"))
	})

	It("should return error when FetchSecretInformation fails", func() {
		delete(parameters, "objectStorageSecretName")

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should return error when secret is not found", func() {
		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to get object store user secret"))
	})

	It("should return error when FetchParameters fails", func() {
		secret.Data = map[string][]byte{}
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("accessKeyID is required"))
	})

	It("should return error when S3 client initialization fails in initializeObjectStorageClient", func() {
		// Create a mock secret in the fake clientset
		secret.Data = map[string][]byte{
			"accessKeyId":     []byte("test-access-key"),
			"secretAccessKey": []byte("test-secret-key"),
			"endpoint":        []byte("https://test-endpoint"),
			"region":          []byte("us-west-2"),
		}
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		// Store the original InitS3Client function
		originalInitS3Client := s3client.InitS3Client
		defer func() { s3client.InitS3Client = originalInitS3Client }()

		// Mock InitS3Client to return an error
		s3client.InitS3Client = func(params util.StorageClientParameters) (*s3client.S3Client, error) {
			return nil, fmt.Errorf("mock S3 client initialization error")
		}

		// Call InitializeClient and check for the expected error
		client, params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(client).To(BeNil())
		Expect(params).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize S3 client"))
	})
})

var _ = Describe("FetchParameters", Ordered, func() {
	var (
		secretData map[string][]byte
	)

	BeforeEach(func() {
		secretData = map[string][]byte{
			"accessKeyId":     []byte("test-access-key"),
			"secretAccessKey": []byte("test-secret-key"),
			"endpoint":        []byte("https://test-endpoint"),
			"region":          []byte("us-west-2"),
			"iamEndpoint":     []byte("https://test-iam-endpoint"),
		}
	})

	It("should successfully fetch S3 parameters when all required fields are present", func() {
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.AccessKeyID).To(Equal("test-access-key"))
		Expect(s3Params.SecretAccessKey).To(Equal("test-secret-key"))
		Expect(s3Params.Endpoint).To(Equal("https://test-endpoint"))
		Expect(s3Params.Region).To(Equal("us-west-2"))
		Expect(s3Params.TLSCert).To(BeNil())
	})

	It("should successfully fetch S3 parameters with TLS certificate", func() {
		secretData["tlsCert"] = []byte("test-tls-cert")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.TLSCert).To(Equal([]byte("test-tls-cert")))
	})

	It("should return error if AccessKey is missing", func() {
		delete(secretData, "accessKeyId")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("accessKeyID is required"))
	})

	It("should return error if SecretKey is missing", func() {
		delete(secretData, "secretAccessKey")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("secretAccessKey is required"))
	})

	It("should return error if Endpoint is missing", func() {
		delete(secretData, "endpoint")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint is required"))
	})

	It("should have seperate IAM endpoint if specified", func() {
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.IAMEndpoint).To(Equal("https://test-iam-endpoint"))
		Expect(s3Params.IAMEndpoint).NotTo(Equal(s3Params.Endpoint))
	})

	It("Should have the same IAM endpoint as the endpoint if not specified", func() {
		delete(secretData, "iamEndpoint")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.IAMEndpoint).To(Equal("https://test-endpoint"))
		Expect(s3Params.IAMEndpoint).To(Equal(s3Params.Endpoint))
	})
})

var _ = Describe("ProvisionerServer DriverGrantBucketAccess", func() {
	var (
		mockIAMClient            *MockIAMClient
		provisioner              *driver.ProvisionerServer
		ctx                      context.Context
		clientset                *fake.Clientset
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)
		bucketName, userName     string
		parameters               map[string]string
		request                  *cosiapi.DriverGrantBucketAccessRequest
		iamParams                *util.StorageClientParameters
	)

	BeforeEach(func() {
		ctx = context.TODO()
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Clientset: clientset,
		}
		bucketName = "test-bucket"
		userName = "test-user"
		parameters = map[string]string{
			"key": "value", // Example parameter; adapt as necessary
		}
		request = &cosiapi.DriverGrantBucketAccessRequest{
			BucketId:   bucketName,
			Name:       userName,
			Parameters: parameters,
		}
		iamParams = &util.StorageClientParameters{
			Endpoint: "https://test-endpoint",
			Region:   "us-west-2",
		}

		// Mock InitializeClient
		originalInitializeClient = driver.InitializeClient
		mockIAMClient = &MockIAMClient{}
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			if service == "IAM" {
				return mockIAMClient, iamParams, nil
			}
			return nil, nil, fmt.Errorf("unsupported service: %s", service)
		}
	})

	AfterEach(func() {
		driver.InitializeClient = originalInitializeClient
	})

	It("should return Internal error when IAM client initialization fails", func() {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock initialization error")
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})

	It("should return Internal error for unsupported client type", func() {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &struct{}{}, iamParams, nil // Return unsupported type
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})

	It("should return Internal error when CreateBucketAccess fails", func() {
		mockIAMClient.CreateBucketAccessFunc = func(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error) {
			return nil, fmt.Errorf("mock failure")
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})
})
