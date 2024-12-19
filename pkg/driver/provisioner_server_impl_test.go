package driver_test

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	iamclient "github.com/scality/cosi-driver/pkg/clients/iam"
	s3client "github.com/scality/cosi-driver/pkg/clients/s3"
	"github.com/scality/cosi-driver/pkg/driver"
	"github.com/scality/cosi-driver/pkg/mock"
	"github.com/scality/cosi-driver/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	bucketv1alpha1 "sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned"
	bucketclientfake "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned/fake"
	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
)

// Test constants
const (
	testProvisionerName = "test-provisioner"
	testBucketName      = "test-bucket"
	testSecretName      = "test-secret"
	testNamespace       = "test-namespace"
	testAccessKey       = "test-access-key"
	testSecretKey       = "test-secret-key"
	testEndpoint        = "https://test-endpoint"
	testRegion          = "us-west-2"
	testIAMEndpoint     = "https://iam-test-endpoint"
)

// Global original references to restore after tests
var (
	originalInClusterConfig     = driver.InClusterConfig
	originalNewKubernetesClient = driver.NewKubernetesClient
	originalNewBucketClient     = driver.NewBucketClient
	originalInitializeClient    = driver.InitializeClient
)

// Helper functions

func setupDefaultConfigMocks() {
	driver.InClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{}, nil
	}
	driver.NewKubernetesClient = func(config *rest.Config) (kubernetes.Interface, error) {
		return fake.NewSimpleClientset(), nil
	}
	driver.NewBucketClient = func(config *rest.Config) (bucketclientset.Interface, error) {
		return bucketclientfake.NewSimpleClientset(), nil
	}
}

func resetConfigMocks() {
	driver.InClusterConfig = originalInClusterConfig
	driver.NewKubernetesClient = originalNewKubernetesClient
	driver.NewBucketClient = originalNewBucketClient
}

func createTestParameters() map[string]string {
	return map[string]string{
		"objectStorageSecretName":      testSecretName,
		"objectStorageSecretNamespace": testNamespace,
	}
}

func createTestSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testSecretName,
			Namespace: testNamespace,
		},
		Data: map[string][]byte{
			"accessKeyId":     []byte(testAccessKey),
			"secretAccessKey": []byte(testSecretKey),
			"endpoint":        []byte(testEndpoint),
			"region":          []byte(testRegion),
		},
	}
}

func createTestSecretDataWithIAMEndpoint() map[string][]byte {
	return map[string][]byte{
		"accessKeyId":     []byte(testAccessKey),
		"secretAccessKey": []byte(testSecretKey),
		"endpoint":        []byte(testEndpoint),
		"region":          []byte(testRegion),
		"iamEndpoint":     []byte(testIAMEndpoint),
	}
}

func createTestS3Params() util.StorageClientParameters {
	return util.StorageClientParameters{
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		Endpoint:        testEndpoint,
		Region:          testRegion,
		IAMEndpoint:     testIAMEndpoint,
	}
}

func createTestIAMParams() util.StorageClientParameters {
	return util.StorageClientParameters{
		AccessKeyID:     testAccessKey,
		SecretAccessKey: testSecretKey,
		Endpoint:        testEndpoint,
		Region:          testRegion,
		IAMEndpoint:     testIAMEndpoint,
	}
}

func createTestProvisionerServer(clientset kubernetes.Interface, bucketClientset bucketclientset.Interface) *driver.ProvisionerServer {
	if clientset == nil {
		clientset = fake.NewSimpleClientset()
	}
	if bucketClientset == nil {
		bucketClientset = bucketclientfake.NewSimpleClientset()
	}
	return &driver.ProvisionerServer{
		Provisioner:     testProvisionerName,
		Clientset:       clientset,
		BucketClientset: bucketClientset,
	}
}

func mockInitializeClient(service string, client interface{}, params *util.StorageClientParameters, err error) {
	driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, s string) (interface{}, *util.StorageClientParameters, error) {
		if s == service {
			return client, params, err
		}
		return nil, nil, fmt.Errorf("unsupported service: %s", s)
	}
}

func restoreInitializeClient() {
	driver.InitializeClient = originalInitializeClient
}

// Tests

var _ = Describe("ProvisionerServer InitProvisionerServer", func() {
	var provisioner string

	BeforeEach(func() {
		setupDefaultConfigMocks()
		provisioner = testProvisionerName
	})

	AfterEach(func() {
		resetConfigMocks()
	})

	It("should initialize a ProvisionerServer successfully", func() {
		server, err := driver.InitProvisionerServer(provisioner)
		Expect(err).To(BeNil())
		Expect(server).NotTo(BeNil())

		ps, ok := server.(*driver.ProvisionerServer)
		Expect(ok).To(BeTrue())
		Expect(ps.Provisioner).To(Equal(provisioner))
		Expect(ps.Clientset).NotTo(BeNil())
		Expect(ps.KubeConfig).NotTo(BeNil())
		Expect(ps.BucketClientset).NotTo(BeNil())
	})

	It("should return error if InClusterConfig fails", func() {
		driver.InClusterConfig = func() (*rest.Config, error) {
			return nil, errors.New("mock error: failed to get in-cluster config")
		}

		server, err := driver.InitProvisionerServer(provisioner)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mock error: failed to get in-cluster config"))
		Expect(server).To(BeNil())
	})

	It("should return error if Kubernetes client creation fails", func() {
		driver.NewKubernetesClient = func(config *rest.Config) (kubernetes.Interface, error) {
			return nil, errors.New("mock error: failed to create Kubernetes client")
		}

		server, err := driver.InitProvisionerServer(provisioner)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mock error: failed to create Kubernetes client"))
		Expect(server).To(BeNil())
	})

	It("should return error if BucketClientset creation fails", func() {
		driver.NewBucketClient = func(config *rest.Config) (bucketclientset.Interface, error) {
			return nil, errors.New("mock error: failed to create BucketClientset")
		}

		server, err := driver.InitProvisionerServer(provisioner)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mock error: failed to create BucketClientset"))
		Expect(server).To(BeNil())
	})

	It("should return error if provisioner name is empty", func() {
		provisioner = ""
		server, err := driver.InitProvisionerServer(provisioner)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("provisioner name cannot be empty"))
		Expect(server).To(BeNil())
	})
})

var _ = Describe("ProvisionerServer DriverCreateBucket", Ordered, func() {
	var (
		mockS3      *mock.MockS3Client
		provisioner *driver.ProvisionerServer
		clientset   *fake.Clientset
		request     *cosiapi.DriverCreateBucketRequest
		s3Params    util.StorageClientParameters
	)

	BeforeEach(func() {
		mockS3 = &mock.MockS3Client{}
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Provisioner: testProvisionerName,
			Clientset:   clientset,
		}

		s3Params = createTestS3Params()
		request = &cosiapi.DriverCreateBucketRequest{Name: testBucketName}
		mockInitializeClient("S3", &s3client.S3Client{S3Service: mockS3}, &s3Params, nil)
	})

	AfterEach(func() {
		restoreInitializeClient()
	})

	It("should successfully create a new bucket", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			Expect(input.Bucket).NotTo(BeNil())
			Expect(*input.Bucket).To(Equal(testBucketName))
			return &s3.CreateBucketOutput{}, nil
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.BucketId).To(Equal(testBucketName))
	})

	It("should return AlreadyExists error if bucket already exists", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &types.BucketAlreadyExists{}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.AlreadyExists))
	})

	It("should return Internal error for other S3 errors", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, errors.New("SomeOtherError")
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should return Internal error when InitializeClient fails", func(ctx SpecContext) {
		mockInitializeClient("S3", nil, nil, fmt.Errorf("mock init error"))

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should return InvalidArgument error for unsupported client type", func(ctx SpecContext) {
		mockInitializeClient("S3", &struct{}{}, &s3Params, nil)

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should handle AWS operation errors", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, _ ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &smithy.OperationError{Err: errors.New("AccessDenied")}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
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
		secretName = testSecretName
		namespace = testNamespace
		os.Unsetenv("POD_NAMESPACE")
	})

	AfterEach(func() {
		os.Unsetenv("POD_NAMESPACE")
	})

	It("should fetch secret name/namespace from parameters", func() {
		parameters["objectStorageSecretName"] = secretName
		parameters["objectStorageSecretNamespace"] = namespace

		name, ns, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(name).To(Equal(secretName))
		Expect(ns).To(Equal(namespace))
	})

	It("should use POD_NAMESPACE if namespace not in parameters", func() {
		parameters["objectStorageSecretName"] = secretName
		os.Setenv("POD_NAMESPACE", namespace)

		name, ns, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(name).To(Equal(secretName))
		Expect(ns).To(Equal(namespace))
	})

	It("should return error if secret name missing", func() {
		parameters["objectStorageSecretNamespace"] = namespace
		_, _, err := driver.FetchSecretInformation(parameters)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should return error if namespace missing and POD_NAMESPACE not set", func() {
		parameters["objectStorageSecretName"] = secretName
		_, _, err := driver.FetchSecretInformation(parameters)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should prioritize parameters namespace over POD_NAMESPACE", func() {
		parameters["objectStorageSecretName"] = secretName
		parameters["objectStorageSecretNamespace"] = namespace
		os.Setenv("POD_NAMESPACE", "env-namespace")

		name, ns, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(name).To(Equal(secretName))
		Expect(ns).To(Equal(namespace))
	})

	It("should return error if both missing", func() {
		_, _, err := driver.FetchSecretInformation(parameters)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})
})

var _ = Describe("initializeObjectStorageClient", Ordered, func() {
	var (
		clientset  *fake.Clientset
		parameters map[string]string
		secret     *corev1.Secret
	)

	BeforeEach(func() {
		clientset = fake.NewSimpleClientset()
		parameters = createTestParameters()
		secret = createTestSecret()
	})

	AfterEach(func() {
		restoreInitializeClient()
	})

	It("should initialize S3 client and parameters", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(BeNil())
		Expect(s3Client).NotTo(BeNil())
		Expect(s3Params.AccessKeyID).To(Equal(testAccessKey))
	})

	It("should return error for unsupported provider", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		_, _, err = driver.InitializeClient(ctx, clientset, parameters, "UnsupportedService")
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if secret not found", func(ctx SpecContext) {
		_, _, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if FetchParameters fails", func(ctx SpecContext) {
		secret.Data = map[string][]byte{}
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		_, _, err = driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should fail if S3 client init fails", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		originalInitS3Client := s3client.InitS3Client
		defer func() { s3client.InitS3Client = originalInitS3Client }()

		s3client.InitS3Client = func(params util.StorageClientParameters) (*s3client.S3Client, error) {
			return nil, fmt.Errorf("mock S3 client error")
		}

		_, _, err = driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should initialize IAM client", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		iamClient, iamParams, err := driver.InitializeClient(ctx, clientset, parameters, "IAM")
		Expect(err).To(BeNil())
		Expect(iamClient).NotTo(BeNil())
		Expect(iamParams.Endpoint).To(Equal(testEndpoint))
	})

	It("should fail if IAM client init fails", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets(testNamespace).Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		originalInitIAMClient := iamclient.InitIAMClient
		defer func() { iamclient.InitIAMClient = originalInitIAMClient }()

		iamclient.InitIAMClient = func(params util.StorageClientParameters) (*iamclient.IAMClient, error) {
			return nil, fmt.Errorf("mock IAM error")
		}

		_, _, err = driver.InitializeClient(ctx, clientset, parameters, "IAM")
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should return error when FetchSecretInformation fails", func(ctx SpecContext) {
		delete(parameters, "objectStorageSecretName")

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})
})

var _ = Describe("FetchParameters", Ordered, func() {
	var secretData map[string][]byte

	BeforeEach(func() {
		secretData = createTestSecretDataWithIAMEndpoint()
	})

	It("should fetch required parameters", func() {
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params.Endpoint).To(Equal(testEndpoint))
	})

	It("should handle TLS cert if present", func() {
		secretData["tlsCert"] = []byte("test-tls-cert")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params.TLSCert).To(Equal([]byte("test-tls-cert")))
	})

	It("should fail if AccessKey missing", func() {
		delete(secretData, "accessKeyId")
		_, err := driver.FetchParameters(secretData)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should fail if SecretKey missing", func() {
		delete(secretData, "secretAccessKey")
		_, err := driver.FetchParameters(secretData)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should fail if Endpoint missing", func() {
		delete(secretData, "endpoint")
		_, err := driver.FetchParameters(secretData)
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should have separate IAM endpoint if specified", func() {
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params.IAMEndpoint).To(Equal(testIAMEndpoint))
	})

	It("should default IAM endpoint to Endpoint if none specified", func() {
		delete(secretData, "iamEndpoint")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params.IAMEndpoint).To(Equal(testEndpoint))
	})
})

var _ = Describe("ProvisionerServer DriverGrantBucketAccess", Ordered, func() {
	var (
		mockIAMClient *mock.MockIAMClient
		provisioner   *driver.ProvisionerServer
		clientset     *fake.Clientset
		request       *cosiapi.DriverGrantBucketAccessRequest
		iamParams     *util.StorageClientParameters
	)

	BeforeEach(func() {
		mockIAMClient = &mock.MockIAMClient{}
		clientset = fake.NewSimpleClientset()
		provisioner = createTestProvisionerServer(clientset, nil)

		iamParamsVal := createTestIAMParams()
		iamParams = &iamParamsVal

		request = &cosiapi.DriverGrantBucketAccessRequest{
			BucketId: testBucketName,
			Name:     "test-user",
		}
		mockInitializeClient("IAM", &iamclient.IAMClient{IAMService: mockIAMClient}, iamParams, nil)
	})

	AfterEach(func() {
		restoreInitializeClient()
	})

	It("should fail if IAM client init fails", func(ctx SpecContext) {
		mockInitializeClient("IAM", nil, nil, fmt.Errorf("init error"))
		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if unsupported client type", func(ctx SpecContext) {
		mockInitializeClient("IAM", &struct{}{}, iamParams, nil)
		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should grant bucket access", func(ctx SpecContext) {
		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp.AccountId).To(Equal("test-user"))
	})

	It("should fail if CreateAccessKey fails", func(ctx SpecContext) {
		mockIAMClient.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, _ ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
			return nil, fmt.Errorf("unable to create access key")
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})
})

var _ = Describe("ProvisionerServer DriverRevokeBucketAccess", Ordered, func() {
	var (
		provisioner   *driver.ProvisionerServer
		mockIAMClient *mock.MockIAMClient
		bucketClient  *bucketclientfake.Clientset
		clientset     *fake.Clientset
		request       *cosiapi.DriverRevokeBucketAccessRequest
	)

	BeforeEach(func() {
		bucketClient = bucketclientfake.NewSimpleClientset()
		clientset = fake.NewSimpleClientset()

		provisioner = createTestProvisionerServer(clientset, bucketClient)
		mockIAMClient = &mock.MockIAMClient{}

		_, err := bucketClient.ObjectstorageV1alpha1().Buckets().Create(context.TODO(), &bucketv1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: testBucketName,
			},
			Spec: bucketv1alpha1.BucketSpec{
				Parameters: createTestParameters(),
			},
		}, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		request = &cosiapi.DriverRevokeBucketAccessRequest{
			BucketId:  testBucketName,
			AccountId: "test-user",
		}

		mockInitializeClient("IAM", &iamclient.IAMClient{IAMService: mockIAMClient}, &util.StorageClientParameters{
			Endpoint:        testEndpoint,
			AccessKeyID:     testAccessKey,
			SecretAccessKey: testSecretKey,
		}, nil)
	})

	AfterEach(func() {
		restoreInitializeClient()
	})

	It("should revoke bucket access", func(ctx SpecContext) {
		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
	})

	It("should fail if bucket does not exist", func(ctx SpecContext) {
		err := bucketClient.ObjectstorageV1alpha1().Buckets().Delete(context.TODO(), testBucketName, metav1.DeleteOptions{})
		Expect(err).To(BeNil())

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if IAM init fails", func(ctx SpecContext) {
		mockInitializeClient("IAM", nil, nil, fmt.Errorf("init error"))
		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail on user deletion error", func(ctx SpecContext) {
		mockIAMClient.GetUserFunc = func(ctx context.Context, input *iam.GetUserInput, _ ...func(*iam.Options)) (*iam.GetUserOutput, error) {
			return nil, &smithy.OperationError{Err: errors.New("AccessDenied")}
		}

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if wrong client type returned", func(ctx SpecContext) {
		mockInitializeClient("IAM", &s3client.S3Client{S3Service: &mock.MockS3Client{}}, nil, nil)
		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})
})

var _ = Describe("ProvisionerServer DriverDeleteBucket", Ordered, func() {
	var (
		mockS3Client *mock.MockS3Client
		provisioner  *driver.ProvisionerServer
		clientset    *fake.Clientset
		bucketClient *bucketclientfake.Clientset
		request      *cosiapi.DriverDeleteBucketRequest
		s3Params     util.StorageClientParameters
	)

	BeforeEach(func() {
		mockS3Client = &mock.MockS3Client{}
		clientset = fake.NewSimpleClientset()
		bucketClient = bucketclientfake.NewSimpleClientset()

		provisioner = createTestProvisionerServer(clientset, bucketClient)
		s3Params = createTestS3Params()

		_, err := bucketClient.ObjectstorageV1alpha1().Buckets().Create(context.TODO(), &bucketv1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: testBucketName,
			},
			Spec: bucketv1alpha1.BucketSpec{
				Parameters: createTestParameters(),
			},
		}, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		request = &cosiapi.DriverDeleteBucketRequest{BucketId: testBucketName}

		mockInitializeClient("S3", &s3client.S3Client{S3Service: mockS3Client}, &s3Params, nil)
	})

	AfterEach(func() {
		restoreInitializeClient()
	})

	It("should delete bucket", func(ctx SpecContext) {
		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
	})

	It("should fail if bucket not found", func(ctx SpecContext) {
		err := bucketClient.ObjectstorageV1alpha1().Buckets().Delete(context.TODO(), testBucketName, metav1.DeleteOptions{})
		Expect(err).To(BeNil())

		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail if S3 init fails", func(ctx SpecContext) {
		mockInitializeClient("S3", nil, nil, fmt.Errorf("init error"))
		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})

	It("should fail on unsupported client type", func(ctx SpecContext) {
		mockInitializeClient("S3", &struct{}{}, &s3Params, nil)
		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
	})

	It("should fail if unable to delete bucket", func(ctx SpecContext) {
		mockS3Client.DeleteBucketFunc = func(ctx context.Context, input *s3.DeleteBucketInput, _ ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
			return nil, fmt.Errorf("unable to delete bucket")
		}

		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
	})
})
