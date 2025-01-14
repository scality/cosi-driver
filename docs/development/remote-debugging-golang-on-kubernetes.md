# Remote Debugging Golang-Based Services on Kubernetes with Delve and VS Code

This guide walks you through setting up a remote debugging environment for Go applications deployed on a Kubernetes cluster using Delve and VS Code. The steps apply to any Go service on Kubernetes, using the Scality COSI driver as an example.

---

## Prerequisites

Ensure the following are installed and configured before starting:

1. **Kubernetes Cluster**: A running Kubernetes cluster, such as [Minikube](https://minikube.sigs.k8s.io/docs/start/), [Docker Desktop Kubernetes](https://docs.docker.com/desktop/features/kubernetes/), or a remote cluster.
2. **kubectl**: [Installed](https://kubernetes.io/docs/tasks/tools/) and configured for your cluster.
3. **VS Code**: [Installed](https://code.visualstudio.com/), with the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go).
4. **COSI Driver**: Clone the repository and navigate to the directory:

   ```bash
   git clone git@github.com:scality/cosi-driver.git && cd cosi-driver
   ```

5. **Delve**: [Installed locally](https://github.com/go-delve/delve/tree/master/Documentation/installation).

---

## Step 1: Build the Container Image

Build the Docker image with Delve by running:

```bash
make delve
```

---

## Step 2: Deploy the COSI Driver

Deploy the COSI driver to Kubernetes using Kustomize. This deployment is configured to run Delve in wait mode, meaning it won’t start the COSI service until a debugger attaches.

```bash
kubectl apply -k kustomize/overlays/debug
```

### Verify the Pod Status

Wait until the pod is ready to ensure the deployment succeeded:

```bash
kubectl wait --namespace container-object-storage-system --for=condition=ready pod --selector=app.kubernetes.io/name=scality-cosi-driver --timeout=120s
```

---

## Step 3: Forward the Delve Debugger Port

Identify the pod name for the COSI driver:

```bash
kubectl get pods -n container-object-storage-system
```

Forward port `2345` from the Kubernetes pod to your local machine to connect VS Code to the Delve debugger:

```bash
kubectl port-forward -n container-object-storage-system pod/<pod-name> 2345:2345
```

Replace `<pod-name>` with the actual name of the pod.

---

## Step 4: Configure VS Code for Remote Debugging

1. Confirm **Delve** is installed locally.
2. Open VS Code and create a `launch.json` file under the `.vscode` directory.
3. Add the following configuration to `launch.json`:

   ```json
   {
       "version": "0.2.0",
       "configurations": [
           {
               "name": "Remote Debug Scality COSI Driver",
               "type": "go",
               "request": "attach",
               "mode": "remote",
               "remotePath": "/app",
               "port": 2345,
               "host": "127.0.0.1",
               "apiVersion": 2,
               "trace": "verbose"
           }
       ]
   }
   ```

---

## Step 6: Start Debugging

1. **Run Port Forwarding**: Ensure port forwarding is active (Step 4).
2. **Initiate Debugging in VS Code**:
   - Open VS Code, set breakpoints in your Go code, and press **F5** to start debugging with the "Remote Debug Scality COSI Driver" configuration.
3. **Inspect Variables and Stack**: You can now inspect variables, step through the code, and debug your Go application as it runs on Kubernetes.

---

## Troubleshooting

- **Delve Not Found**: Verify Delve is installed in your Docker image and available at `/dlv`.
- **Port Forwarding Issues**: Confirm the Kubernetes pod is running, and that port `2345` is open.
- **Breakpoints Not Hit**: Ensure the code in Kubernetes matches the local code in VS Code.
- **Connection Timeout**: Check firewall rules, network policies, and Kubernetes pod permissions if the debugger cannot connect.
