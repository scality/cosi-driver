# Running COSI Driver Locally

This guide walks you through the steps to run the Scality COSI driver locally using Minikube and Unix socket for development purposes. Follow the instructions below to set up and verify the COSI driver in your local environment.

## Prerequisites

Ensure the following are installed and configured before starting:

1. **Minikube**: [Installed](https://minikube.sigs.k8s.io/docs/start/) and running.
2. **kubectl**: [Installed](https://kubernetes.io/docs/tasks/tools/) and configured to use Minikube or your desired Kubernetes cluster.
3. **Go**: [Installed](https://golang.org/doc/install) to build the required Go tools.
4. **gRPCurl**: [Installed](https://github.com/fullstorydev/grpcurl#installation) for interacting with the COSI driver over gRPC.

## Step 1: Apply Kubernetes Resources

To deploy the necessary Kubernetes resources for the COSI driver, run:

```bash
kubectl apply -k kustomize/overlays/dev
```

## Step 2: Emulate the Service Account

COSI requires a service account, which can be emulated by creating the appropriate directories and copying configuration files. Run the following commands:

```sh
# Create the required directories for the service account
sudo mkdir -p /var/run/secrets/kubernetes.io/serviceaccount/

# Create the service account token file
sudo touch /var/run/secrets/kubernetes.io/serviceaccount/token

# Copy the Minikube kubeconfig file to the service account token
sudo cp ~/.kube/config /var/run/secrets/kubernetes.io/serviceaccount/token

# Copy the Minikube CA certificate
sudo cp ~/.minikube/ca.crt /var/run/secrets/kubernetes.io/serviceaccount/
```

## Step 3: Start the COSI Driver

Now, start the COSI driver on the Unix socket path /var/lib/cosi/cosi.sock. Ensure that the KUBERNETES_SERVICE_HOST is set to your Minikube IP address, and run the following:

```sh
KUBERNETES_SERVICE_HOST=$(minikube ip) KUBERNETES_SERVICE_PORT=6443 ./bin/scality-cosi-driver --driver-address unix://$(pwd)/cosi.sock

# Output
I1119 20:18:58.211369   68342 cmd.go:52] "COSI driver startup configuration" driverAddress="unix:///path/to/cosi.sock" driverPrefix="cosi"
```

## Step 4: Verify Socket Creation

To verify that the socket file has been created, run:

```sh
ls ./cosi.sock
```

You should see the cosi.sock file listed.

## Step 5: Query the COSI Driver

Now, you can use grpcurl to interact with the COSI driver. First, list the available services:

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock list

# Output
cosi.v1alpha1.Identity
cosi.v1alpha1.Provisioner
```

You can also list methods for a specific service. For example, to list methods for the Identity service:

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock list cosi.v1alpha1.Identity

# Output
cosi.v1alpha1.Identity.DriverGetInfo
```

Similarly, list methods for the Provisioner service:

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock list cosi.v1alpha1.Provisioner

# Output
cosi.v1alpha1.Provisioner.DriverCreateBucket
cosi.v1alpha1.Provisioner.DriverDeleteBucket
cosi.v1alpha1.Provisioner.DriverGrantBucketAccess
cosi.v1alpha1.Provisioner.DriverRevokeBucketAccess
```

It is also possible to describe the APIs

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock describe cosi.v1alpha1.Provisioner.DriverCreateBucket

#Output
cosi.v1alpha1.Provisioner.DriverCreateBucket is a method:
// This call is made to create the bucket in the backend.
// This call is idempotent
//    1. If a bucket that matches both name and parameters already exists, then OK (success) must be returned.
//    2. If a bucket by same name, but different parameters is provided, then the appropriate error code ALREADY_EXISTS must be returned.
rpc DriverCreateBucket ( .cosi.v1alpha1.DriverCreateBucketRequest ) returns ( .cosi.v1alpha1.DriverCreateBucketResponse );
```

Step 8: Invoke Methods on the COSI Driver

To invoke methods on the COSI driver, you can use grpcurl with a JSON payload. For example,

- DriverGetInfo gRPC API

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix ./cosi.sock cosi.v1alpha1.Identity.DriverGetInfo

# Output
{
  "name": "cosi.scality.com"
}
```

- DriverCreateBucket gRPC API

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix -d '{
  "name": "example-bucket",
  "parameters": {
    "objectStorageSecretName": "s3-secret-for-cosi",
    "objectStorageSecretNamespace": "default"
  }
}' ./cosi.sock cosi.v1alpha1.Provisioner.DriverCreateBucket
```

- DriverGrantBucketAccess gRPC API

```sh
grpcurl -plaintext -proto cosi.proto -import-path ./proto -unix -d '{
  "name": "user-name",
  "bucketId": "example-bucket",
  "parameters": {
    "objectStorageSecretName": "s3-secret-for-cosi",
    "objectStorageSecretNamespace": "default"
  }
}' ./cosi.sock cosi.v1alpha1.Provisioner.DriverGrantBucketAccess
```

## Troubleshooting

- Socket Not Found: If /var/lib/cosi/cosi.sock is not created, ensure the COSI driver started correctly by checking its logs.
- gRPCurl Errors: If grpcurl fails, ensure the cosi.proto file is in the correct location and the socket is accessible.
- Connection Issues: If there are issues connecting to Minikube, verify the Minikube cluster is running (minikube status) and that the correct IP is being used.
