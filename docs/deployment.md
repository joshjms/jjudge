# Deploying JJudge on Kubernetes

This guide covers a production deployment on a managed Kubernetes cluster (GKE, EKS, or AKS) with KEDA autoscaling workers based on RabbitMQ queue wait time.

## Architecture overview

```
                        ┌─────────────────────────────────────┐
                        │           Kubernetes cluster         │
                        │                                      │
  Users ──► Ingress ──► │  frontend   apiserver   grader       │
                        │                 │                    │
                        │           RabbitMQ                   │
                        │          submissions                 │
                        │       contest-submissions            │
                        │               │                      │
                        │    ┌──────────▼──────────┐          │
                        │    │   worker Deployment  │          │
                        │    │   (KEDA autoscaled)  │          │
                        │    │  min 1 – max N pods  │          │
                        │    └─────────────────────-┘          │
                        │                                      │
                        │  Postgres    MinIO    Prometheus      │
                        └─────────────────────────────────────┘
```

The worker requires `privileged: true` and host cgroup access for lime's sandboxing. Workers should run on **dedicated judge nodes** with a taint so no other workloads land there.

---

## Prerequisites

| Tool | Version |
|------|---------|
| kubectl | ≥ 1.28 |
| helm | ≥ 3.13 |
| Docker | ≥ 24 |
| A container registry | (GCR, ECR, GHCR, etc.) |

---

## 1. Dedicated judge node pool

Workers need privileged access and CPU isolation. Create a separate node pool tainted to reserve it exclusively for judge workloads.

**GKE example:**
```sh
gcloud container node-pools create judge-pool \
  --cluster <your-cluster> \
  --machine-type n2-standard-8 \
  --num-nodes 1 \
  --node-taints dedicated=judge:NoSchedule \
  --node-labels dedicated=judge \
  --enable-autoscaling --min-nodes 1 --max-nodes 10
```

**EKS example (`eksctl`):**
```yaml
# nodegroup in cluster.yaml
- name: judge-pool
  instanceType: c6i.2xlarge
  minSize: 1
  maxSize: 10
  labels:
    dedicated: judge
  taints:
    - key: dedicated
      value: judge
      effect: NoSchedule
```

Workers schedule onto these nodes via `nodeSelector` + `tolerations` (see step 5).

---

## 2. Build and push images

```sh
export REGISTRY=ghcr.io/<your-org>
export TAG=$(git rev-parse --short HEAD)

# Build from the repo root (context must be the monorepo root)
docker build -f worker/Dockerfile    -t $REGISTRY/jjudge-worker:$TAG    .
docker build -f apiserver/Dockerfile -t $REGISTRY/jjudge-apiserver:$TAG .
docker build -f grader/Dockerfile    -t $REGISTRY/jjudge-grader:$TAG    ./grader
docker build -f frontend/Dockerfile  -t $REGISTRY/jjudge-frontend:$TAG  ./frontend \
  --build-arg NEXT_PUBLIC_API_BASE_URL=https://api.example.com

docker push $REGISTRY/jjudge-worker:$TAG
docker push $REGISTRY/jjudge-apiserver:$TAG
docker push $REGISTRY/jjudge-grader:$TAG
docker push $REGISTRY/jjudge-frontend:$TAG
```

---

## 3. Namespace and secrets

```sh
kubectl create namespace jjudge
```

Create secrets for every credential. Do not put secrets in manifests.

```sh
# Postgres
kubectl -n jjudge create secret generic postgres-secret \
  --from-literal=password=<strong-password>

# MinIO / S3
kubectl -n jjudge create secret generic minio-secret \
  --from-literal=access-key=<access-key> \
  --from-literal=secret-key=<secret-key>

# RabbitMQ
kubectl -n jjudge create secret generic rabbitmq-secret \
  --from-literal=password=<strong-password> \
  --from-literal=url=amqp://jjudge:<strong-password>@rabbitmq.jjudge.svc:5672/

# apiserver
kubectl -n jjudge create secret generic apiserver-secret \
  --from-literal=jwt-secret=<random-256-bit-string> \
  --from-literal=admin-password=<admin-password>
```

---

## 4. Infrastructure services

### RabbitMQ

Install via the official Helm chart. Enable the Prometheus plugin so KEDA can query queue metrics.

```sh
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

helm install rabbitmq bitnami/rabbitmq \
  --namespace jjudge \
  --set auth.username=jjudge \
  --set auth.existingPasswordSecret=rabbitmq-secret \
  --set auth.existingSecretPasswordKey=password \
  --set metrics.enabled=true \
  --set extraPlugins="rabbitmq_prometheus" \
  --set persistence.size=8Gi
```

### PostgreSQL

```sh
helm install postgres bitnami/postgresql \
  --namespace jjudge \
  --set auth.username=jjudge \
  --set auth.database=jjudge \
  --set auth.existingSecret=postgres-secret \
  --set auth.secretKeys.userPasswordKey=password \
  --set primary.persistence.size=20Gi
```

### MinIO

```sh
helm repo add minio https://charts.min.io/

helm install minio minio/minio \
  --namespace jjudge \
  --set rootUser=minioadmin \
  --set existingSecret=minio-secret \
  --set persistence.size=50Gi \
  --set buckets[0].name=jjudge,buckets[0].policy=none
```

---

## 5. Application deployments

### grader

```yaml
# grader.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: grader
  namespace: jjudge
spec:
  replicas: 2
  selector:
    matchLabels:
      app: grader
  template:
    metadata:
      labels:
        app: grader
    spec:
      containers:
        - name: grader
          image: ghcr.io/<your-org>/jjudge-grader:<tag>
          ports:
            - containerPort: 50051
          env:
            - name: SERVER_PORT
              value: "50051"
---
apiVersion: v1
kind: Service
metadata:
  name: grader
  namespace: jjudge
spec:
  selector:
    app: grader
  ports:
    - port: 50051
```

### apiserver

```yaml
# apiserver.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apiserver
  namespace: jjudge
spec:
  replicas: 2
  selector:
    matchLabels:
      app: apiserver
  template:
    metadata:
      labels:
        app: apiserver
    spec:
      containers:
        - name: apiserver
          image: ghcr.io/<your-org>/jjudge-apiserver:<tag>
          ports:
            - containerPort: 8080
          env:
            - name: SERVER_PORT
              value: "8080"
            - name: DB_HOST
              value: postgres-postgresql.jjudge.svc
            - name: DB_PORT
              value: "5432"
            - name: DB_USER
              value: jjudge
            - name: DB_NAME
              value: jjudge
            - name: DB_USE_SSL
              value: "false"
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: apiserver-secret
                  key: jwt-secret
            - name: JJUDGE_ADMIN_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: apiserver-secret
                  key: admin-password
            - name: MINIO_ENDPOINT
              value: minio.jjudge.svc:9000
            - name: MINIO_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: access-key
            - name: MINIO_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: secret-key
            - name: MINIO_BUCKET
              value: jjudge
            - name: RABBITMQ_URL
              valueFrom:
                secretKeyRef:
                  name: rabbitmq-secret
                  key: url
---
apiVersion: v1
kind: Service
metadata:
  name: apiserver
  namespace: jjudge
spec:
  selector:
    app: apiserver
  ports:
    - port: 8080
```

### worker

The worker needs:
- `privileged: true` — for lime's user namespaces and OverlayFS
- `cgroupns: host` — to access the host cgroup v2 tree
- Node selector + toleration — to land on judge nodes only
- One pod per node is sufficient; KEDA adds nodes via cluster autoscaler, not more pods per node (see step 6)

```yaml
# worker.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker
  namespace: jjudge
spec:
  replicas: 1
  selector:
    matchLabels:
      app: worker
      queue: submissions
  template:
    metadata:
      labels:
        app: worker
        queue: submissions
    spec:
      # Pin to dedicated judge nodes
      nodeSelector:
        dedicated: judge
      tolerations:
        - key: dedicated
          operator: Equal
          value: judge
          effect: NoSchedule

      # One worker pod per node to avoid CPU contention between sandboxes
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels:
              app: worker

      containers:
        - name: worker
          image: ghcr.io/<your-org>/jjudge-worker:<tag>
          securityContext:
            privileged: true
          env:
            - name: GRADER_ADDR
              value: grader.jjudge.svc:50051
            - name: JUDGE_SUBMISSIONS_DIR
              value: /tmp/judge/submissions
            - name: JUDGE_WORK_ROOT
              value: /tmp/judge/work
            - name: JUDGE_OVERLAYFS_DIR
              value: /tmp/judge/overlayfs
            - name: JUDGE_ROOTFS_DIR
              value: /rootfs
            - name: JUDGE_CPUS
              value: "0-3"         # adjust to node size
            - name: LIME_CGROUP_ROOT
              value: /sys/fs/cgroup/lime.slice
            - name: RABBITMQ_URL
              valueFrom:
                secretKeyRef:
                  name: rabbitmq-secret
                  key: url
            - name: RABBITMQ_QUEUE
              value: submissions
            - name: MINIO_ENDPOINT
              value: minio.jjudge.svc:9000
            - name: MINIO_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: access-key
            - name: MINIO_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: minio-secret
                  key: secret-key
            - name: MINIO_BUCKET
              value: jjudge
          volumeMounts:
            - name: cgroup
              mountPath: /sys/fs/cgroup
      volumes:
        - name: cgroup
          hostPath:
            path: /sys/fs/cgroup
            type: Directory
```

Deploy the contest worker the same way, changing `RABBITMQ_QUEUE: contest-submissions` and `JUDGE_CPUS: "4-7"` (or whatever cores are free on the node).

---

## 6. KEDA autoscaling

KEDA scales the worker `Deployment` by adding or removing pods (which causes the cluster autoscaler to provision or deprovision judge nodes).

### Install KEDA

```sh
helm repo add kedacore https://kedacore.github.io/charts
helm repo update

helm install keda kedacore/keda \
  --namespace keda \
  --create-namespace
```

### How queue wait time is measured

RabbitMQ's Prometheus exporter exposes `rabbitmq_queue_messages_ready` (depth) and `rabbitmq_queue_messages_delivered_total` (throughput). Average queue wait time is:

```
wait_time_seconds = messages_ready / deliver_rate
```

KEDA's Prometheus scaler evaluates this expression and scales when the result exceeds a threshold.

### Install kube-prometheus-stack (for the Prometheus datasource)

```sh
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false
```

Add a `ServiceMonitor` to scrape RabbitMQ:

```yaml
# rabbitmq-servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: rabbitmq
  namespace: jjudge
  labels:
    release: prometheus     # must match the Helm release label
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: rabbitmq
  endpoints:
    - port: metrics
      path: /metrics
      interval: 15s
```

### ScaledObject for the submissions worker

```yaml
# worker-scaledobject.yaml
apiVersion: keda.sh/v1alpha1
kind: ScaledObject
metadata:
  name: worker-scaledobject
  namespace: jjudge
spec:
  scaleTargetRef:
    name: worker

  # Keep at least one worker running at all times
  minReplicaCount: 1
  maxReplicaCount: 20

  # How long to wait before scaling down an idle worker
  cooldownPeriod: 120     # seconds

  # Poll the trigger every 15 seconds
  pollingInterval: 15

  triggers:
    - type: prometheus
      metadata:
        serverAddress: http://prometheus-kube-prometheus-prometheus.monitoring.svc:9090
        metricName: rabbitmq_submissions_wait_time_seconds
        # Scale out when the average submission has been waiting more than 30 seconds
        threshold: "30"
        query: |
          rabbitmq_queue_messages_ready{queue="submissions"}
          /
          clamp_min(
            rate(rabbitmq_queue_messages_delivered_total{queue="submissions"}[2m]),
            0.01
          )
```

`clamp_min(..., 0.01)` prevents division by zero when no messages have been delivered yet (idle state keeps the value low so we don't spuriously scale up).

Deploy a matching `ScaledObject` for `worker-contest` pointing at the `contest-submissions` queue.

### Verify scaling

```sh
# Watch KEDA events
kubectl -n jjudge describe scaledobject worker-scaledobject

# Watch pod count change as queue fills
kubectl -n jjudge get pods -l app=worker -w

# Manually publish messages to test (requires rabbitmqadmin or the management UI)
```

---

## 7. Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: jjudge
  namespace: jjudge
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.example.com
        - example.com
      secretName: jjudge-tls
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: apiserver
                port:
                  number: 8080
    - host: example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend
                port:
                  number: 3000
```

---

## 8. Database migrations

Run migrations as a `Job` before (or as an init container of) the apiserver deployment:

```yaml
# migrate-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: apiserver-migrate
  namespace: jjudge
spec:
  template:
    spec:
      restartPolicy: OnFailure
      containers:
        - name: migrate
          image: ghcr.io/<your-org>/jjudge-apiserver:<tag>
          command: ["/usr/local/bin/apiserver", "migrate", "up"]
          env:
            - name: DB_HOST
              value: postgres-postgresql.jjudge.svc
            - name: DB_PORT
              value: "5432"
            - name: DB_USER
              value: jjudge
            - name: DB_NAME
              value: jjudge
            - name: DB_USE_SSL
              value: "false"
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgres-secret
                  key: password
```

```sh
kubectl apply -f migrate-job.yaml
kubectl -n jjudge wait --for=condition=complete job/apiserver-migrate --timeout=120s
```

---

## 9. Apply all manifests

```sh
kubectl apply -f migrate-job.yaml
kubectl apply -f grader.yaml
kubectl apply -f apiserver.yaml
kubectl apply -f worker.yaml
kubectl apply -f rabbitmq-servicemonitor.yaml
kubectl apply -f worker-scaledobject.yaml
kubectl apply -f ingress.yaml
```

---

## Scaling behaviour summary

| Condition | KEDA action | Cluster autoscaler action |
|-----------|-------------|--------------------------|
| Queue wait time > 30 s | Increase worker replica count | Provision a new judge node if no capacity |
| Queue wait time < 30 s for 120 s | Decrease worker replica count | Deprovision idle judge node |
| Queue empty, 1 replica remaining | Hold at `minReplicaCount: 1` | No change |
