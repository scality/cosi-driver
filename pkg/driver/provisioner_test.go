package driver_test

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/scality/cosi/pkg/driver"
	s3client "github.com/scality/cosi/pkg/util/s3client"
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

var _ = Describe("ProvisionerServer DriverCreateBucket", func() {
	var (
		mockS3                   *MockS3Client
		provisioner              *driver.ProvisionerServer
		ctx                      context.Context
		clientset                *fake.Clientset
		bucketName               string
		s3Params                 s3client.S3Params
		request                  *cosiapi.DriverCreateBucketRequest
		originalInitializeClient func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string) (*s3client.S3Client, *s3client.S3Params, error)
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
		s3Params = s3client.S3Params{
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
			Endpoint:  "https://test-endpoint",
			Region:    "us-west-2",
		}
		request = &cosiapi.DriverCreateBucketRequest{Name: bucketName}

		// Storing the original method to restore it for other test suits
		originalInitializeClient = driver.InitializeClient
	})

	AfterEach(func() {
		// Restore the original InitializeClient function for other test suites
		driver.InitializeClient = originalInitializeClient
	})

	JustBeforeEach(func() {
		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string) (*s3client.S3Client, *s3client.S3Params, error) {
			return &s3client.S3Client{S3Service: mockS3}, &s3Params, nil
		}
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
})

var _ = Describe("ProvisionerServer Unimplemented Methods", func() {
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

	It("DriverGrantBucketAccess should return Unimplemented error", func() {
		request := &cosiapi.DriverGrantBucketAccessRequest{
			BucketId: bucketName,
			Name:     "test-access",
		}
		resp, err := provisioner.DriverGrantBucketAccess(ctx, request)
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

var _ = Describe("FetchSecretInformation", func() {
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
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME"] = secretName
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAMESPACE"] = namespace

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(fetchedSecretName).To(Equal(secretName))
		Expect(fetchedNamespace).To(Equal(namespace))
	})

	It("should use POD_NAMESPACE environment variable when namespace is not in parameters", func() {
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME"] = secretName
		os.Setenv("POD_NAMESPACE", namespace)

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(BeNil())
		Expect(fetchedSecretName).To(Equal(secretName))
		Expect(fetchedNamespace).To(Equal(namespace))
	})

	It("should return error when secret name is missing", func() {
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAMESPACE"] = namespace

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(HaveOccurred())
		Expect(fetchedSecretName).To(BeEmpty())
		Expect(fetchedNamespace).To(BeEmpty())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should return error when namespace is missing and POD_NAMESPACE is not set", func() {
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME"] = secretName

		fetchedSecretName, fetchedNamespace, err := driver.FetchSecretInformation(parameters)
		Expect(err).To(HaveOccurred())
		Expect(fetchedSecretName).To(BeEmpty())
		Expect(fetchedNamespace).To(BeEmpty())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should prioritize namespace from parameters over POD_NAMESPACE environment variable", func() {
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME"] = secretName
		parameters["COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAMESPACE"] = namespace
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

var _ = Describe("initializeObjectStorageClient", func() {
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
			"COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME":      "test-secret",
			"COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAMESPACE": "test-namespace",
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "test-namespace",
			},
			Data: map[string][]byte{
				"COSI_S3_ACCESS_KEY_ID":     []byte("test-access-key"),
				"COSI_S3_SECRET_ACCESS_KEY": []byte("test-secret-key"),
				"COSI_S3_ENDPOINT":          []byte("https://test-endpoint"),
				"COSI_S3_REGION":            []byte("us-west-2"),
			},
		}
	})

	It("should successfully initialize S3 client and parameters", func() {
		_, err := clientset.CoreV1().Secrets("test-namespace").Create(ctx, secret, metav1.CreateOptions{})
		Expect(err).To(BeNil())

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters)
		Expect(err).To(BeNil())
		Expect(s3Client).NotTo(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.AccessKey).To(Equal("test-access-key"))
		Expect(s3Params.SecretKey).To(Equal("test-secret-key"))
		Expect(s3Params.Endpoint).To(Equal("https://test-endpoint"))
		Expect(s3Params.Region).To(Equal("us-west-2"))
	})

	It("should return error when FetchSecretInformation fails", func() {
		delete(parameters, "COSI_OBJECT_STORAGE_PROVIDER_SECRET_NAME")

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters)
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("Object storage provider secret name and namespace are required"))
	})

	It("should return error when secret is not found", func() {
		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters)
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

		s3Client, s3Params, err := driver.InitializeClient(ctx, clientset, parameters)
		Expect(err).To(HaveOccurred())
		Expect(s3Client).To(BeNil())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint, accessKeyID, secretKey and region are required"))
	})
})

var _ = Describe("FetchParameters", func() {
	var (
		secretData map[string][]byte
	)

	BeforeEach(func() {
		secretData = map[string][]byte{
			"COSI_S3_ACCESS_KEY_ID":     []byte("test-access-key"),
			"COSI_S3_SECRET_ACCESS_KEY": []byte("test-secret-key"),
			"COSI_S3_ENDPOINT":          []byte("https://test-endpoint"),
			"COSI_S3_REGION":            []byte("us-west-2"),
		}
	})

	It("should successfully fetch S3 parameters when all required fields are present", func() {
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.AccessKey).To(Equal("test-access-key"))
		Expect(s3Params.SecretKey).To(Equal("test-secret-key"))
		Expect(s3Params.Endpoint).To(Equal("https://test-endpoint"))
		Expect(s3Params.Region).To(Equal("us-west-2"))
		Expect(s3Params.TLSCert).To(BeNil())
	})

	It("should successfully fetch S3 parameters with TLS certificate", func() {
		secretData["COSI_S3_TLS_CERT_SECRET_NAME"] = []byte("test-tls-cert")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(BeNil())
		Expect(s3Params).NotTo(BeNil())
		Expect(s3Params.TLSCert).To(Equal([]byte("test-tls-cert")))
	})

	It("should return error if AccessKey is missing", func() {
		delete(secretData, "COSI_S3_ACCESS_KEY_ID")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint, accessKeyID, secretKey and region are required"))
	})

	It("should return error if SecretKey is missing", func() {
		delete(secretData, "COSI_S3_SECRET_ACCESS_KEY")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint, accessKeyID, secretKey and region are required"))
	})

	It("should return error if Endpoint is missing", func() {
		delete(secretData, "COSI_S3_ENDPOINT")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint, accessKeyID, secretKey and region are required"))
	})

	It("should return error if Region is missing", func() {
		delete(secretData, "COSI_S3_REGION")
		s3Params, err := driver.FetchParameters(secretData)
		Expect(err).To(HaveOccurred())
		Expect(s3Params).To(BeNil())
		Expect(status.Code(err)).To(Equal(codes.InvalidArgument))
		Expect(err.Error()).To(ContainSubstring("endpoint, accessKeyID, secretKey and region are required"))
	})
})
