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

var _ = Describe("ProvisionerServer InitProvisionerServer", func() {
	var (
		provisioner string
	)

	BeforeEach(func() {
		driver.InClusterConfig = func() (*rest.Config, error) {
			return &rest.Config{}, nil
		}

		driver.NewKubernetesClient = func(config *rest.Config) (kubernetes.Interface, error) {
			return fake.NewSimpleClientset(), nil
		}

		driver.NewBucketClient = func(config *rest.Config) (bucketclientset.Interface, error) {
			return bucketclientfake.NewSimpleClientset(), nil
		}

		provisioner = "test-provisioner"
	})

	AfterEach(func() {
		// Restore original functions
		driver.InClusterConfig = rest.InClusterConfig
		driver.NewKubernetesClient = func(c *rest.Config) (kubernetes.Interface, error) {
			return kubernetes.NewForConfig(c)
		}
		driver.NewBucketClient = func(c *rest.Config) (bucketclientset.Interface, error) {
			return bucketclientset.NewForConfig(c)
		}
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
		mockS3                   *mock.MockS3Client
		provisioner              *driver.ProvisionerServer
		clientset                *fake.Clientset
		bucketName               string
		s3Params                 util.StorageClientParameters
		request                  *cosiapi.DriverCreateBucketRequest
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)
	)

	BeforeEach(func() {
		mockS3 = &mock.MockS3Client{}
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

	It("should successfully create a new bucket", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			Expect(input.Bucket).To(Equal(&bucketName))
			return &s3.CreateBucketOutput{}, nil
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.BucketId).To(Equal(bucketName))
	})

	It("should return AlreadyExists error if bucket already exists with different parameters", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &types.BucketAlreadyExists{}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.AlreadyExists))
		Expect(err.Error()).To(ContainSubstring("Bucket already exists: test-bucket"))
	})

	It("should return success if bucket with same parameters already exists", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &types.BucketAlreadyOwnedByYou{}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.BucketId).To(Equal(bucketName))
	})

	It("should return Internal error for other S3 client errors", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, errors.New("SomeOtherError: Something went wrong")
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("Failed to create bucket"))
	})

	It("should return Internal error when InitializeClient fails", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock initialization error")
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider S3 client"))
	})

	It("should return InvalidArgument error when client type is unsupported", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &struct{}{}, &s3Params, nil
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("unsupported client type for bucket creation"))
	})

	It("should handle AWS operation errors during bucket creation", func(ctx SpecContext) {
		mockS3.CreateBucketFunc = func(ctx context.Context, input *s3.CreateBucketInput, opts ...func(*s3.Options)) (*s3.CreateBucketOutput, error) {
			return nil, &smithy.OperationError{
				Err: errors.New("AccessDenied: Access Denied"),
			}
		}

		resp, err := provisioner.DriverCreateBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("Failed to create bucket"))
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
		clientset  *fake.Clientset
		parameters map[string]string
		secret     *corev1.Secret
	)

	BeforeEach(func() {
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

	It("should successfully initialize S3 client and parameters", func(ctx SpecContext) {
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

	It("should return InvalidArgument error for unsupported object storage provider service", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		client, params, err := driver.InitializeClient(ctx, clientset, parameters, "UnsupportedService")

		Expect(client).To(BeNil())
		Expect(params).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("unsupported object storage provider service"))
	})

	It("should return error when FetchSecretInformation fails", func(ctx SpecContext) {
		delete(parameters, "objectStorageSecretName")

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should return error when secret is not found", func(ctx SpecContext) {
		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to get object store user secret"))
	})

	It("should return error when FetchParameters fails", func(ctx SpecContext) {
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

	It("should return error when S3 client initialization fails in initializeObjectStorageClient", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		originalInitS3Client := s3client.InitS3Client
		defer func() { s3client.InitS3Client = originalInitS3Client }()

		s3client.InitS3Client = func(params util.StorageClientParameters) (*s3client.S3Client, error) {
			return nil, fmt.Errorf("mock S3 client initialization error")
		}

		client, params, err := driver.InitializeClient(ctx, clientset, parameters, "S3")
		Expect(client).To(BeNil())
		Expect(params).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize S3 client"))
	})

	It("should successfully initialize IAM client", func(ctx SpecContext) {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		iamClient, iamParams, err := driver.InitializeClient(ctx, clientset, parameters, "IAM")
		Expect(err).To(BeNil())
		Expect(iamClient).NotTo(BeNil())
		Expect(iamParams.Endpoint).To(Equal("https://test-endpoint"))
		Expect(iamParams.AccessKeyID).To(Equal("test-access-key"))
		Expect(iamParams.SecretAccessKey).To(Equal("test-secret-key"))
		Expect(iamParams.Region).To(Equal("us-west-2"))
	})

	It("should return Internal error when IAM client initialization fails", func(ctx SpecContext) {
		originalInitIAMClient := iamclient.InitIAMClient
		defer func() { iamclient.InitIAMClient = originalInitIAMClient }()

		iamclient.InitIAMClient = func(params util.StorageClientParameters) (*iamclient.IAMClient, error) {
			return nil, fmt.Errorf("mock IAM client initialization error")
		}

		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		iamClient, iamParams, err := driver.InitializeClient(ctx, clientset, parameters, "IAM")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("failed to initialize IAM client")))
		Expect(iamClient).To(BeNil())
		Expect(iamParams).To(BeNil())
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

var _ = Describe("ProvisionerServer DriverGrantBucketAccess", Ordered, func() {
	var (
		mockIAMClient            *mock.MockIAMClient
		provisioner              *driver.ProvisionerServer
		clientset                *fake.Clientset
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)
		bucketName, userName     string
		request                  *cosiapi.DriverGrantBucketAccessRequest
		iamParams                *util.StorageClientParameters
	)

	BeforeEach(func() {
		mockIAMClient = &mock.MockIAMClient{}
		clientset = fake.NewSimpleClientset()
		provisioner = &driver.ProvisionerServer{
			Provisioner: "test-provisioner",
			Clientset:   clientset,
		}
		bucketName = "test-bucket"
		userName = "test-user"

		iamParams = &util.StorageClientParameters{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Endpoint:        "https://test-endpoint",
			Region:          "us-west-2",
			IAMEndpoint:     "https://iam-test-endpoint",
		}
		request = &cosiapi.DriverGrantBucketAccessRequest{
			BucketId: bucketName,
			Name:     userName,
		}
		originalInitializeClient = driver.InitializeClient

		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			if service == "IAM" {
				return &iamclient.IAMClient{IAMService: mockIAMClient}, iamParams, nil
			}
			return nil, nil, fmt.Errorf("unsupported service: %s", service)
		}
	})

	AfterEach(func() {
		driver.InitializeClient = originalInitializeClient
	})

	It("should return Internal error when IAM client initialization fails", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock initialization error")
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})

	It("should return Internal error for unsupported client type", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &struct{}{}, iamParams, nil
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})

	It("should successfully grant bucket access", func(ctx SpecContext) {
		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp.AccountId).To(Equal(userName))
		Expect(resp.Credentials["s3"].Secrets["accessKeyID"]).To(Equal("mock-access-key-id"))
		Expect(resp.Credentials["s3"].Secrets["accessSecretKey"]).To(Equal("mock-secret-access-key"))
		Expect(resp.Credentials["s3"].Secrets["endpoint"]).To(Equal(iamParams.Endpoint))
		Expect(resp.Credentials["s3"].Secrets["region"]).To(Equal(iamParams.Region))
	})

	It("should return Internal error when CreateBucketAccess fails", func(ctx SpecContext) {
		mockIAMClient.CreateAccessKeyFunc = func(ctx context.Context, input *iam.CreateAccessKeyInput, opts ...func(*iam.Options)) (*iam.CreateAccessKeyOutput, error) {
			return nil, fmt.Errorf("mock failure: unable to create access key")
		}

		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to create bucket access"))
	})
})

var _ = Describe("ProvisionerServer DriverRevokeBucketAccess", Ordered, func() {
	var (
		provisioner              *driver.ProvisionerServer
		mockIAMClient            *mock.MockIAMClient
		bucketClientset          *bucketclientfake.Clientset
		clientset                *fake.Clientset
		bucketName, userName     string
		request                  *cosiapi.DriverRevokeBucketAccessRequest
		secretName               string
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)

		namespace string
	)

	BeforeEach(func() {
		// Initialize fake clients
		bucketClientset = bucketclientfake.NewSimpleClientset()
		clientset = fake.NewSimpleClientset()

		// Initialize ProvisionerServer with the fake BucketClientset
		provisioner = &driver.ProvisionerServer{
			Provisioner:     "test-provisioner",
			Clientset:       clientset,
			BucketClientset: bucketClientset,
		}
		mockIAMClient = &mock.MockIAMClient{}

		bucketName = "test-bucket"
		userName = "test-user"
		secretName = "my-storage-secret"
		namespace = "test-namespace"

		// Create a fake Bucket object with appropriate parameters
		_, err := bucketClientset.ObjectstorageV1alpha1().Buckets().Create(context.TODO(), &bucketv1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: bucketName,
			},
			Spec: bucketv1alpha1.BucketSpec{
				Parameters: map[string]string{
					"objectStorageSecretName":      secretName,
					"objectStorageSecretNamespace": namespace,
				},
			},
		}, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		// Create the request
		request = &cosiapi.DriverRevokeBucketAccessRequest{
			BucketId:  bucketName,
			AccountId: userName,
		}
		originalInitializeClient = driver.InitializeClient

		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			// Validate parameters
			Expect(parameters["objectStorageSecretName"]).To(Equal(secretName))
			Expect(parameters["objectStorageSecretNamespace"]).To(Equal(namespace))

			if service == "IAM" {
				return &iamclient.IAMClient{IAMService: mockIAMClient}, &util.StorageClientParameters{
					Endpoint:        "https://test-endpoint",
					AccessKeyID:     "test-access-key",
					SecretAccessKey: "test-secret-key",
				}, nil
			}
			return nil, nil, errors.New("unsupported service")
		}
	})

	AfterEach(func() {
		mockIAMClient = nil
		bucketClientset = nil
		clientset = nil
		provisioner = nil
		driver.InitializeClient = originalInitializeClient
	})

	It("should successfully revoke bucket access when bucket exists and parameters are valid", func(ctx SpecContext) {
		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)

		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp).To(BeAssignableToTypeOf(&cosiapi.DriverRevokeBucketAccessResponse{}))
	})

	It("should return error if the bucket does not exist", func(ctx SpecContext) {
		err := bucketClientset.ObjectstorageV1alpha1().Buckets().Delete(context.TODO(), bucketName, metav1.DeleteOptions{})
		Expect(err).To(BeNil())

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)

		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to get bucket object from kubernetes"))
	})

	It("should return error if IAM client initialization fails", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock IAM client initialization error")
		}

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)

		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider IAM client"))
	})

	It("should fail if unable to delete user", func(ctx SpecContext) {
		mockIAMClient.GetUserFunc = func(ctx context.Context, input *iam.GetUserInput, opts ...func(*iam.Options)) (*iam.GetUserOutput, error) {
			return nil, &smithy.OperationError{
				Err: errors.New("AccessDenied: Access Denied"),
			}
		}

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to revoke bucket access"))
	})

	It("should fail if unable to the right client", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &s3client.S3Client{S3Service: &mock.MockS3Client{}}, &util.StorageClientParameters{
				Endpoint:        "https://test-endpoint",
				AccessKeyID:     "test-access-key",
				SecretAccessKey: "test-secret-key",
			}, nil
		}

		resp, err := provisioner.DriverRevokeBucketAccess(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("unsupported client type for IAM operations"))
	})
})

var _ = Describe("ProvisionerServer DriverDeleteBucket", Ordered, func() {
	var (
		mockS3Client             *mock.MockS3Client
		provisioner              *driver.ProvisionerServer
		clientset                *fake.Clientset
		bucketName               string
		request                  *cosiapi.DriverDeleteBucketRequest
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error)
		bucketClientset          *bucketclientfake.Clientset
		secretName, namespace    string
		s3Params                 util.StorageClientParameters
	)

	BeforeEach(func() {
		mockS3Client = &mock.MockS3Client{}
		clientset = fake.NewSimpleClientset()
		bucketClientset = bucketclientfake.NewSimpleClientset()

		provisioner = &driver.ProvisionerServer{
			Provisioner:     "test-provisioner",
			Clientset:       clientset,
			BucketClientset: bucketClientset,
		}

		s3Params = util.StorageClientParameters{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			Endpoint:        "https://test-endpoint",
			Region:          "us-west-2",
		}
		secretName = "my-storage-secret"
		namespace = "test-namespace"

		// Create a fake Bucket object with appropriate parameters
		_, err := bucketClientset.ObjectstorageV1alpha1().Buckets().Create(context.TODO(), &bucketv1alpha1.Bucket{
			ObjectMeta: metav1.ObjectMeta{
				Name: bucketName,
			},
			Spec: bucketv1alpha1.BucketSpec{
				Parameters: map[string]string{
					"objectStorageSecretName":      secretName,
					"objectStorageSecretNamespace": namespace,
				},
			},
		}, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		request = &cosiapi.DriverDeleteBucketRequest{BucketId: bucketName}
		originalInitializeClient = driver.InitializeClient
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			// Validate parameters
			Expect(parameters["objectStorageSecretName"]).To(Equal(secretName))
			Expect(parameters["objectStorageSecretNamespace"]).To(Equal(namespace))

			if service == "S3" {
				return &s3client.S3Client{S3Service: mockS3Client}, &s3Params, nil
			}
			return nil, nil, errors.New("unsupported service")
		}
	})

	AfterEach(func() {
		mockS3Client = nil
		bucketClientset = nil
		clientset = nil
		provisioner = nil
		driver.InitializeClient = originalInitializeClient
	})

	It("should successfully delete bucket when bucket exists and parameters are valid", func(ctx SpecContext) {
		resp, err := provisioner.DriverDeleteBucket(ctx, request)

		Expect(err).To(BeNil())
		Expect(resp).NotTo(BeNil())
		Expect(resp).To(BeAssignableToTypeOf(&cosiapi.DriverDeleteBucketResponse{}))
	})

	It("should return error if the bucket does not exist", func(ctx SpecContext) {
		err := bucketClientset.ObjectstorageV1alpha1().Buckets().Delete(context.TODO(), bucketName, metav1.DeleteOptions{})
		Expect(err).To(BeNil())

		resp, err := provisioner.DriverDeleteBucket(ctx, request)

		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to get bucket object from kubernetes"))
	})

	It("should return error if S3 client initialization fails", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return nil, nil, fmt.Errorf("mock S3 client initialization error")
		}

		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to initialize object storage provider S3 client"))
	})

	It("should return InvalidArgument error for unsupported client type", func(ctx SpecContext) {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
			return &struct{}{}, &s3Params, nil
		}

		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("unsupported client type for bucket deletion"))
	})

	It("should return error if unable to delete bucket", func(ctx SpecContext) {
		mockS3Client.DeleteBucketFunc = func(ctx context.Context, input *s3.DeleteBucketInput, opts ...func(*s3.Options)) (*s3.DeleteBucketOutput, error) {
			return nil, fmt.Errorf("mock failure: unable to delete bucket")
		}

		resp, err := provisioner.DriverDeleteBucket(ctx, request)
		Expect(resp).To(BeNil())
		Expect(err).To(HaveOccurred())
		Expect(status.Code(err)).To(Equal(codes.Internal))
		Expect(err.Error()).To(ContainSubstring("failed to delete bucket"))
	})
})
