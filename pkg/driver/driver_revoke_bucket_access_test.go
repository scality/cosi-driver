package driver_test

// import (
// 	"context"
// 	"fmt"

// 	. "github.com/onsi/ginkgo/v2"
// 	. "github.com/onsi/gomega"
// 	"github.com/scality/cosi-driver/pkg/driver"
// 	"github.com/scality/cosi-driver/pkg/util"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/runtime"
// 	"k8s.io/client-go/kubernetes"
// 	"k8s.io/client-go/kubernetes/fake"
// 	k8stesting "k8s.io/client-go/testing"
// 	bucketv1alpha1 "sigs.k8s.io/container-object-storage-interface-api/apis/objectstorage/v1alpha1"
// 	bucketclientset "sigs.k8s.io/container-object-storage-interface-api/client/clientset/versioned/fake"
// 	cosiapi "sigs.k8s.io/container-object-storage-interface-spec"
// )

// func (m *MockIAMClient) RevokeBucketAccess(ctx context.Context, userName, bucketName string) error {
// 	if m.RevokeBucketAccessFunc != nil {
// 		return m.RevokeBucketAccessFunc(ctx, userName, bucketName)
// 	}
// 	return nil
// }

// var _ = Describe("ProvisionerServer DriverRevokeBucketAccess", func() {
// 	var (
// 		provisioner          *driver.ProvisionerServer
// 		ctx                  context.Context
// 		mockIAMClient        *MockIAMClient
// 		originalInitClient   func(context.Context, kubernetes.Interface, map[string]string, string) (interface{}, *util.StorageClientParameters, error)
// 		bucketName, userName string
// 		iamParams            *util.StorageClientParameters
// 	)

// 	BeforeEach(func() {
// 		ctx = context.TODO()
// 		mockIAMClient = &MockIAMClient{}

// 		// Mock InitializeClient
// 		originalInitClient = driver.InitializeClient
// 		driver.InitializeClient = func(ctx context.Context, clientset kubernetes.Interface, parameters map[string]string, service string) (interface{}, *util.StorageClientParameters, error) {
// 			if service == "IAM" {
// 				return mockIAMClient, iamParams, nil
// 			}
// 			return nil, nil, fmt.Errorf("unsupported service: %s", service)
// 		}

// 		// Mock BucketClientset with a test bucket
// 		bucket := &bucketv1alpha1.Bucket{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      "test-bucket",
// 				Namespace: "default",
// 			},
// 			Spec: bucketv1alpha1.BucketSpec{
// 				Parameters: map[string]string{
// 					"objectStorageSecretName":      "s3-secret-for-cosi",
// 					"objectStorageSecretNamespace": "default",
// 				},
// 			},
// 		}
// 		bucketClientset := bucketclientset.NewSimpleClientset(bucket)
// 		bucketClientset.Fake.PrependReactor("get", "buckets", func(action k8stesting.Action) (bool, runtime.Object, error) {
// 			getAction := action.(k8stesting.GetAction)
// 			if getAction.GetName() == bucket.Name {
// 				return true, bucket, nil
// 			}
// 			return true, nil, fmt.Errorf("bucket not found")
// 		})

// 		provisioner = &driver.ProvisionerServer{
// 			Clientset:       fake.NewSimpleClientset(),
// 			BucketClientset: bucketClientset,
// 		}

// 		bucketName = "test-bucket"
// 		userName = "test-user"
// 		iamParams = &util.StorageClientParameters{
// 			Endpoint: "https://test-iam-endpoint",
// 			Region:   "us-west-2",
// 		}
// 	})

// 	AfterEach(func() {
// 		driver.InitializeClient = originalInitClient
// 	})

// 	It("should successfully revoke bucket access", func() {
// 		mockIAMClient.RevokeBucketAccessFunc = func(ctx context.Context, userName, bucketName string) error {
// 			if userName == "invalid-user" {
// 				return fmt.Errorf("user not found")
// 			}
// 			return nil
// 		}

// 		resp, err := provisioner.DriverRevokeBucketAccess(ctx, &cosiapi.DriverRevokeBucketAccessRequest{
// 			BucketId:  bucketName,
// 			AccountId: userName,
// 		})
// 		Expect(err).To(BeNil())
// 		Expect(resp).To(BeAssignableToTypeOf(&cosiapi.DriverRevokeBucketAccessResponse{}))
// 	})
// })
