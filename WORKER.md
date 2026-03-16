# Deploying jjudge-worker on Kubernetes (Scalable)

This document describes how to deploy `jjudge-worker` on a Kubernetes cluster with
auto-scaling support. Workers run `lime`, a rootless container runtime that requires
privileged host access to cgroups, user namespaces, and OverlayFS.

---

## Prerequisites

- A Kubernetes cluster (GKE, EKS, k3s, etc.) with cgroup v2 enabled on nodes
- `kubectl` and `helm` installed and configured
- A container registry to push built images
- RabbitMQ accessible from the cluster
- [KEDA](https://keda.sh) installed in the cluster (for queue-based pod autoscaling)
- [Cluster Autoscaler](https://github.com/kubernetes/autoscaler) configured for your node pool

---

## Step 1: Build a Custom Node Image

New worker nodes must have the following pre-installed before any pod runs. This cannot
be done at pod startup time.

**Required on every worker node:**

1. `newuidmap` and `newgidmap` (from `uidmap` / `shadow-utils` package)
2. `/etc/subuid` and `/etc/subgid` entries for UID 1000 (the worker process user):
   ```
   # /etc/subuid
   1000:100000:65536

   # /etc/subgid
   1000:100000:65536
   ```
3. OverlayFS kernel module loaded (`modprobe overlay`)
4. cgroup v2 enabled (verify: `stat -fc %T /sys/fs/cgroup` should print `cgroup2fs`)

Build this into a custom machine image (AMI, GCE image, etc.) for your node pool.
Use cloud-init or a packer template to apply these changes on top of a base OS image.

---

## Step 2: Create a Dedicated Worker Node Pool

Create a node pool (or node group) separate from your general workloads.

**Required node pool configuration:**

- **Taint**: `jjudge=worker:NoSchedule` — prevents non-worker pods from landing here
- **Label**: `jjudge/role=worker` — used by pod `nodeSelector`
- **Autoscaling**: enabled, with a defined min/max node count (e.g., min: 1, max: 10)
- **Machine type**: choose based on memory and CPU requirements of your problem set

Example (GKE):
```bash
gcloud container node-pools create worker-pool \
  --cluster=<your-cluster> \
  --machine-type=n2-standard-4 \
  --num-nodes=1 \
  --min-nodes=1 \
  --max-nodes=10 \
  --enable-autoscaling \
  --node-taints=jjudge=worker:NoSchedule \
  --node-labels=jjudge/role=worker \
  --image-type=UBUNTU_CONTAINERD
```

---

## Step 3: Deploy the cgroup Initializer DaemonSet

Each worker node needs `/sys/fs/cgroup/lime.slice` created and ownership delegated to
UID 1000 before any worker pod runs. A `DaemonSet` with a privileged init container
handles this automatically whenever a new node joins the pool.

```yaml
# k8s/worker-cgroup-init-daemonset.yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: worker-cgroup-init
  namespace: jjudge
spec:
  selector:
    matchLabels:
      app: worker-cgroup-init
  template:
    metadata:
      labels:
        app: worker-cgroup-init
    spec:
      nodeSelector:
        jjudge/role: worker
      tolerations:
        - key: jjudge
          operator: Equal
          value: worker
          effect: NoSchedule
      initContainers:
        - name: cgroup-setup
          image: busybox
          command:
            - sh
            - -c
            - |
              mkdir -p /sys/fs/cgroup/lime.slice
              chown 1000:1000 /sys/fs/cgroup/lime.slice
              chown 1000:1000 /sys/fs/cgroup/lime.slice/cgroup.procs
              chown 1000:1000 /sys/fs/cgroup/lime.slice/cgroup.subtree_control
          securityContext:
            privileged: true
          volumeMounts:
            - name: cgroup
              mountPath: /sys/fs/cgroup
      containers:
        - name: pause
          image: gcr.io/google_containers/pause:3.9
      volumes:
        - name: cgroup
          hostPath:
            path: /sys/fs/cgroup
```

Apply:
```bash
kubectl apply -f k8s/worker-cgroup-init-daemonset.yaml
```

---

## Step 4: Build and Push the Worker Image

The worker image must include the `lime` binary and the compiled `jjudge-worker` binary.

```dockerfile
# jjudge-worker/Dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o /worker .

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y uidmap && rm -rf /var/lib/apt/lists/*
COPY --from=builder /worker /usr/local/bin/worker
COPY --from=<lime-build> /lime/build/lime /usr/local/bin/lime
RUN useradd -u 1000 -m judge
USER 1000
ENTRYPOINT ["/usr/local/bin/worker"]
```

Build and push:
```bash
docker build -t <your-registry>/jjudge-worker:latest ./jjudge-worker
docker push <your-registry>/jjudge-worker:latest
```

---

## Step 5: Deploy the Worker

```yaml
# k8s/worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jjudge-worker
  namespace: jjudge
spec:
  replicas: 1  # KEDA will override this
  selector:
    matchLabels:
      app: jjudge-worker
  template:
    metadata:
      labels:
        app: jjudge-worker
    spec:
      nodeSelector:
        jjudge/role: worker
      tolerations:
        - key: jjudge
          operator: Equal
          value: worker
          effect: NoSchedule
      # Ensure cgroup init DaemonSet pod is ready before workers start
      initContainers:
        - name: wait-for-cgroup-init
          image: busybox
          command:
            - sh
            - -c
            - |
              until [ -d /sys/fs/cgroup/lime.slice ]; do
                echo "waiting for lime.slice..."; sleep 1;
              done
          volumeMounts:
            - name: cgroup
              mountPath: /sys/fs/cgroup
              readOnly: true
      containers:
        - name: worker
          image: <your-registry>/jjudge-worker:latest
          securityContext:
            runAsUser: 1000
            runAsGroup: 1000
            capabilities:
              add:
                - SYS_ADMIN  # required for user namespaces and overlayfs
          env:
            - name: ENV
              value: "prod"
            - name: LIME_CGROUP_ROOT
              value: /sys/fs/cgroup/lime.slice
            - name: JUDGE_MAX_CONCURRENCY
              value: "4"
            - name: RABBITMQ_URL
              valueFrom:
                secretKeyRef:
                  name: jjudge-secrets
                  key: rabbitmq-url
            # Add remaining env vars from CLAUDE.md here
          volumeMounts:
            - name: cgroup
              mountPath: /sys/fs/cgroup
            - name: work-root
              mountPath: /judge/work
          resources:
            requests:
              cpu: "1"
              memory: "512Mi"
            limits:
              cpu: "2"
              memory: "2Gi"
      volumes:
        - name: cgroup
          hostPath:
            path: /sys/fs/cgroup
        - name: work-root
          emptyDir: {}
```

Apply:
```bash
kubectl apply -f k8s/worker-deployment.yaml
```

---

## Step 6: Install KEDA and Configure Queue-Based Autoscaling

[KEDA](https://keda.sh) scales the worker `Deployment` based on RabbitMQ queue depth.

Install KEDA:
```bash
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace
```

Create a `ScaledObject` that watches the submissions queue:
```yaml
# k8s/worker-scaledobject.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: jjudge-worker-scaler
  namespace: jjudge
spec:
  scaleTargetRef:
    name: jjudge-worker
  minReplicaCount: 0   # scale to zero when queue is empty
  maxReplicaCount: 20  # hard cap
  pollingInterval: 10  # seconds between queue checks
  cooldownPeriod: 30   # seconds before scaling down
  triggers:
    - type: rabbitmq
      metadata:
        host: amqp://<rabbitmq-host>:5672/
        queueName: submissions
        mode: QueueLength
        value: "5"  # one pod per 5 queued jobs
      authenticationRef:
        name: rabbitmq-auth
---
apiVersion: keda.sh/v1alpha1
kind: TriggerAuthentication
metadata:
  name: rabbitmq-auth
  namespace: jjudge
spec:
  secretTargetRef:
    - parameter: host
      name: jjudge-secrets
      key: rabbitmq-url
```

Apply:
```bash
kubectl apply -f k8s/worker-scaledobject.yaml
```

---

## Step 7: Verify Cluster Autoscaler

Cluster Autoscaler scales the **node pool** when pods cannot be scheduled due to
insufficient nodes. Verify it is watching the worker node pool.

For GKE, autoscaling is enabled per node pool (Step 2). For other providers, ensure
the Cluster Autoscaler deployment includes the worker node group in its configuration.

To verify it is working, watch for `ScaleUp` events:
```bash
kubectl get events -n kube-system | grep -i scale
```

---

## Scaling Summary

| Layer | Tool | Trigger |
|---|---|---|
| Pods | KEDA | RabbitMQ queue depth |
| Nodes | Cluster Autoscaler | Pending pods (insufficient node capacity) |

When a contest starts and submissions flood in:
1. KEDA detects queue depth rising → adds worker pods
2. New pods are `Pending` because existing nodes are full → Cluster Autoscaler adds nodes
3. New nodes run the cgroup-init DaemonSet → worker pods become `Running`

When the queue drains:
1. KEDA scales pods down to 0
2. Cluster Autoscaler detects underutilized nodes → drains and removes them

---

## Security Notes

- Worker pods require `SYS_ADMIN` capability for user namespaces and OverlayFS. Isolate
  the worker node pool with network policies to limit blast radius.
- Do not run any other workloads on worker nodes (enforced by the taint).
- Consider using a dedicated Kubernetes service account with minimal RBAC for the worker.
