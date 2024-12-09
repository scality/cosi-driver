package driver_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	iamclient "github.com/scality/cosi-driver/pkg/clients/iam"
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	"github.com/scality/cosi-driver/pkg/driver"
	"github.com/scality/cosi-driver/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	bucketv1alpha1 "sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned/fake"
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
	CreateUserFunc         func(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error)
	PutUserPolicyFunc      func(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error)
	CreateAccessKeyFunc    func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error)
	GetUserFunc            func(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error)
	DeleteUserPolicyFunc   func(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error)
	ListAccessKeysFunc     func(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error)
	DeleteAccessKeyFunc    func(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error)
	DeleteUserFunc         func(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error)
	CreateBucketAccessFunc func(ctx context.Context, userName, bucketName string) (*iam.CreateAccessKeyOutput, error)
	RevokeBucketAccessFunc func(ctx context.Context, userName, bucketName string) error
}

// Implement CreateUser
func (m *MockIAMClient) CreateUser(ctx context.Context, input *iam.CreateUserInput, opts ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, input, opts...)
	}
	return &iam.CreateUserOutput{
		User: &iamtypes.User{
			UserName: input.UserName,
			UserId:   aws.String("mock-user-id"),
		},
	}, nil
}

func (m *MockIAMClient) RevokeBucketAccess(ctx context.Context, userName, bucketName string) error {
	if m.RevokeBucketAccessFunc != nil {
		return m.RevokeBucketAccessFunc(ctx, userName, bucketName)
	}
	return nil
}

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

// Implement PutUserPolicy
func (m *MockIAMClient) PutUserPolicy(ctx context.Context, input *iam.PutUserPolicyInput, opts ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
	if m.PutUserPolicyFunc != nil {
		return m.PutUserPolicyFunc(ctx, input, opts...)
	}
	return &iam.PutUserPolicyOutput{}, nil
}

// Implement CreateAccessKey
func (m *MockIAMClient) CreateAccessKey(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
	if m.CreateAccessKeyFunc != nil {
		return m.CreateAccessKeyFunc(ctx, input, opts...)
	}
	return &iam.CreateAccessKeyOutput{
		AccessKey: &iamtypes.AccessKey{
			AccessKeyId:     aws.String("mock-access-key-id"),
			SecretAccessKey: aws.String("mock-secret-access-key"),
		},
	}, nil
}

// Implement GetUser
func (m *MockIAMClient) GetUser(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error) {
	if m.GetUserFunc != nil {
		return m.GetUserFunc(ctx, input, opts...)
	}
	return &iam.GetUserOutput{
		User: &iamtypes.User{
			UserName: input.UserName,
			UserId:   aws.String("mock-user-id"),
		},
	}, nil
}

// Implement DeleteUserPolicy
func (m *MockIAMClient) DeleteUserPolicy(ctx context.Context, input *iam.DeleteUserPolicyInput, opts ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
	if m.DeleteUserPolicyFunc != nil {
		return m.DeleteUserPolicyFunc(ctx, input, opts...)
	}
	return &iam.DeleteUserPolicyOutput{}, nil
}

// Implement ListAccessKeys
func (m *MockIAMClient) ListAccessKeys(ctx context.Context, input *iam.ListAccessKeysInput, opts ...func(*iam.Options)) (*iam.ListAccessKeysOutput, error) {
	if m.ListAccessKeysFunc != nil {
		return m.ListAccessKeysFunc(ctx, input, opts...)
	}
	return &iam.ListAccessKeysOutput{
		AccessKeyMetadata: []iamtypes.AccessKeyMetadata{
			{
				AccessKeyId: aws.String("mock-access-key-id"),
			},
		},
	}, nil
}

// Implement DeleteAccessKey
func (m *MockIAMClient) DeleteAccessKey(ctx context.Context, input *iam.DeleteAccessKeyInput, opts ...func(*iam.Options)) (*iam.DeleteAccessKeyOutput, error) {
	if m.DeleteAccessKeyFunc != nil {
		return m.DeleteAccessKeyFunc(ctx, input, opts...)
	}
	return &iam.DeleteAccessKeyOutput{}, nil
}

// Implement DeleteUser
func (m *MockIAMClient) DeleteUser(ctx context.Context, input *iam.DeleteUserInput, opts ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
	if m.DeleteUserFunc != nil {
		return m.DeleteUserFunc(ctx, input, opts...)
	}
	return &iam.DeleteUserOutput{}, nil
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
	)

	BeforeEach(func() {
		ctx = context.TODO()
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Provisioner: "test-provisioner",
			Clientset:   clientset,
		}
		bucketName = "test-bucket"
	})

	It("DriverDeleteBucket should return Unimplemented error", func() {
		request := &cosiapi.DriverDeleteBucketRequest{BucketId: bucketName}
		resp, err := provisioner.DriverDeleteBucket(ctx, request)
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

	It("should return InvalidArgument error for unsupported object storage provider service", func() {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		client, params, err := driver.InitializeClient(ctx, clientset, parameters, "UnsupportedService")

		Expect(client).To(BeNil())
		Expect(params).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("unsupported object storage provider service"))
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

var _ = Describe("ProvisionerServer DriverRevokeBucketAccess", func() {
	var (
		provisioner          *driver.ProvisionerServer
		ctx                  context.Context
		mockIAMClient        *MockIAMClient
		originalInitClient   func(context.Context, kubernetes.Interface, map[string]string, string) (interface{}, *util.StorageClientParameters, error)
		bucketName, userName string
		iamParams            *util.StorageClientParameters
	)

	BeforeEach(func() {
		ctx = context.TODO()
		mockIAMClient = &MockIAMClient{}

		// Mock InitializeClient
		originalInitClient = driver.InitializeClient
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			if service == "IAM" {
				return &iamclient.IAMClient{
					IAMService: mockIAMClient,
				}, iamParams, nil
			}
			return nil, nil, fmt.Errorf("unsupported service: %s", service)
		}

		// Mock BucketClientset with a test bucket
		bucket := &bucketv1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-bucket",
				Namespace: "default",
			},
			Spec: bucketv1alpha1.BucketSpec{
				Parameters: map[string]string{
					"objectStorageSecretName":      "s3-secret-for-cosi",
					"objectStorageSecretNamespace": "default",
				},
			},
		}
		bucketClientset := bucketclientset.NewSimpleClientset(bucket)
		bucketClientset.Fake.PrependReactor("get", "buckets", func(action k8stesting.Action) (bool, runtime.Object, error) {
			getAction := action.(k8stesting.GetAction)
			if getAction.GetName() == bucket.Name {
				return true, bucket, nil
			}
			return true, nil, fmt.Errorf("bucket not found")
		})

		provisioner = &driver.ProvisionerServer{
			Clientset:       fake.NewSimpleClientset(),
			BucketClientset: bucketClientset,
		}

		bucketName = "test-bucket"
		userName = "test-user"
		iamParams = &util.StorageClientParameters{
			Endpoint: "https://test-iam-endpoint",
			Region:   "us-west-2",
		}
	})

	AfterEach(func() {
		driver.InitializeClient = originalInitClient
	})

	It("should successfully revoke bucket access", func() {
		mockIAMClient.RevokeBucketAccessFunc = func(ctx context.Context, userName, bucketName string) error {
			if userName == "invalid-user" {
				return fmt.Errorf("user not found")
			}
			return nil
		}

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, &cosiapi.DriverRevokeBucketAccessRequest{
			BucketId:  bucketName,
			AccountId: userName,
		})
		Expect(err).To(BeNil())
		Expect(resp).To(BeAssignableToTypeOf(&cosiapi.DriverRevokeBucketAccessResponse{}))
	})
})
