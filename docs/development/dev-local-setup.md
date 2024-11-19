kubectl apply -k kustomize/overlays/dev
# emulate service account
sudo mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/
sudo touch /var/run/secrets/kubernetes.io/serviceaccount/token
sudo cp ~/.kube/config /var/run/secrets/kubernetes.io/serviceaccount/token
sudo cp /Users/anurag4dsb/.minikube/ca.crt /var/run/secrets/kubernetes.io/serviceaccount/


# Start Cosi driver on unix sock path  /var/lib/cosi/cosi.sock
 KUBERNETES_SERVICE_HOST=$(minikube ip) KUBERNETES_SERVICE_PORT=6443 ./bin/scality-cosi-driver  --driver-address unix://$(pwd)cosi.sock 
W1119 20:18:58.211253   68342 cmd.go:49] No driver prefix provided, using default prefix
I1119 20:18:58.211369   68342 cmd.go:52] "COSI driver startup configuration" driverAddress="unix:///Users/anurag4dsb/hyperbolical-time-chamber/scality/cosi-drivercosi.sock" driverPrefix="cosi"              

Verify that new file is created
ls /var/lib/cosi/cosi.sock

Install grpccurl
 go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

grpcurl -plaintext -proto cosi.proto -import-path ./proto  -unix ./cosi.sock list 
cosi.v1alpha1.Identity
cosi.v1alpha1.Provisioner

grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock list cosi.v1alpha1.Identity
cosi.v1alpha1.Identity.DriverGetInfo
 grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock list cosi.v1alpha1.Provisioner
cosi.v1alpha1.Provisioner.DriverCreateBucket
cosi.v1alpha1.Provisioner.DriverDeleteBucket
cosi.v1alpha1.Provisioner.DriverGrantBucketAccess
cosi.v1alpha1.Provisioner.DriverRevokeBucketAccess

 grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock cosi.v1alpha1.Identity.DriverGetInfo
{
  "name": "cosi.scality.com"
}
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock describe cosi.v1alpha1.Provisioner.DriverCreateBucket

cosi.v1alpha1.Provisioner.DriverCreateBucket is a method:
// This call is made to create the bucket in the backend.
// This call is idempotent
//    1. If a bucket that matches both name and parameters already exists, then OK (success) must be returned.
//    2. If a bucket by same name, but different parameters is provided, then the appropriate error code ALREADY_EXISTS must be returned.
rpc DriverCreateBucket ( .cosi.v1alpha1.DriverCreateBucketRequest ) returns ( .cosi.v1alpha1.DriverCreateBucketResponse );


grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix  -d '{
  "name": "example-bucket",
  "parameters": {
    "storageClass": "STANDARD",
    "region": "us-west-1"
  }
}'  ./cosi.sock   cosi.v1alpha1.Provisioner.DriverCreateBucket


grpcurl -plaintext -d '{
  "name": "example-bucket",
  "parameters": {
    "storageClass": "STANDARD",
    "region": "us-west-1"
  }
}'  -proto cosi.proto -import-path ./proto -unix ./cosi.sock cosi.v1alpha1.Provisioner.DriverCreateBucket
ERROR:
  Code: Internal