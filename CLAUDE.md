# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

JJudge is an online judge system (competitive programming platform) consisting of:
- **jjudge**: Next.js 15 frontend (React 19, TypeScript, TailwindCSS 4)
- **jjudge-apiserver**: Go REST API backend (chi router, PostgreSQL, JWT auth)
- **jjudge-worker**: Go worker service for executing code submissions
- **jjudge-grader**: Go gRPC service for grading test outputs
- **jjudge-api**: Shared protobuf definitions and Go types
- **lime**: C-based rootless container runtime for isolated code execution

## System Architecture

### Request Flow
1. User submits code via **jjudge** frontend
2. **jjudge-apiserver** stores submission in PostgreSQL, publishes `SubmissionJob` to RabbitMQ
3. **jjudge-worker** consumes job, compiles code in a lime container, executes against each test case
4. Worker calls **jjudge-grader** via gRPC to compare outputs, then publishes results to a result queue
5. API server consumes results queue and updates submission verdict in database
6. Frontend polls the submission endpoint for verdict updates

### Inter-service Communication
| From → To | Protocol |
|-----------|----------|
| Frontend → API Server | HTTP/REST |
| API Server ↔ Database | PostgreSQL |
| API Server ↔ Storage | MinIO (dev) / GCS (prod) |
| API Server → MQ | RabbitMQ (dev) / GCP Pub/Sub (prod) |
| Worker → MQ | RabbitMQ — consumes `submissions`, publishes `submission-results` |
| Worker → Storage | MinIO/GCS — fetches test case files |
| Worker → Grader | gRPC |
| Worker → Lime | JSON via stdin/stdout |

Contest submissions use separate queues: `contest-submissions` → `contest-submission-results`.

## Development Commands

### All-in-one (recommended for dev)
```bash
./start.sh   # docker-compose up for backend, then npm run dev for frontend
```

### Frontend (jjudge)
```bash
cd jjudge
npm run dev          # Dev server with Turbopack on :3000
npm run build        # Production build
npm run lint         # ESLint
```
Copy `.env.local.example` to `.env.local`; set `NEXT_PUBLIC_API_BASE_URL` (default: `http://localhost:8080`).

### Backend (jjudge-apiserver)
```bash
cd jjudge-apiserver
docker compose up --build              # All services: postgres, minio, rabbitmq, apiserver, grader, 2x worker
go run main.go server                  # API server only
go run main.go migrate up              # Apply migrations
go run main.go migrate down            # Rollback migrations
go test ./internal/tests/e2e/...       # E2E tests (requires docker-compose deps running)
```

### Worker (jjudge-worker)
```bash
cd jjudge-worker
go run main.go                         # Run worker
go test ./internal/lime/...            # Lime integration tests
docker build -t jjudge-worker .        # Docker build (also compiles lime inside)
```

### Grader (jjudge-grader)
```bash
cd jjudge-grader
go run main.go   # gRPC server on :50051
```

### Lime
```bash
cd lime
make              # Build to build/lime (gcc, C2x, -O2 -pthread)
make clean
./build/lime run < config.json
```
Requires: Linux cgroup v2, `newuidmap`/`newgidmap` in PATH, delegated cgroup subtree.

```bash
export LIME_CGROUP_ROOT=/sys/fs/cgroup/user.slice/user-1000.slice/user@1000.service/lime.slice
mkdir -p "$LIME_CGROUP_ROOT"
echo "+cpu +cpuset +memory +pids +io" > /sys/fs/cgroup/cgroup.subtree_control
```

## Code Structure

### Frontend (`jjudge/src`)
- `app/` — Next.js app router pages (admin, contests, problems, submissions, login, register, profile)
- `components/` — React components using Radix UI / shadcn/ui patterns
- `lib/api.ts` — API client wrapper; use `api.get<T>()` / `api.post<T>()` throughout

### API Server (`jjudge-apiserver/internal`)
- `server/` — chi router setup and middleware
- `handlers/` — HTTP handlers: auth, problem, submission, contest
- `services/` — Business logic layer (problem, submission, testcase, user, contest)
- `db/migrations/` — 4 SQL migrations (init → versioning → remove bundles → contests)
- `db/services/` — Database implementations
- `mq/` — RabbitMQ / GCP Pub/Sub abstraction
- `storage/` — MinIO / GCS abstraction
- `store/` — In-memory repository implementations
- `tests/e2e/` — End-to-end tests

Entry point: `cmd/server.go` (Cobra CLI with `server` and `migrate` subcommands).
On startup, apiserver auto-creates an admin user if not present (`JJUDGE_ADMIN_USER` / `JJUDGE_ADMIN_PASSWORD`).

### Worker (`jjudge-worker/internal`)
- `worker/` — `worker.go` wires dependencies; `processor.go` handles job lifecycle
- `lime/` — Lime integration (`lime.go`), slot pool for concurrency (`allocator.go`), result parsing (`report.go`)
- `blob/` — Object storage client for fetching test case files
- `tccache/` — LRU cache for test case files (reduces storage round-trips)
- `mq/` — MQ client
- `grader/` — gRPC client stub for grader service

### Shared Types (`jjudge-api`)
- `proto/` — `.proto` definitions for Submission, Problem, User
- `jjudgepb/` — Generated `*.pb.go` files
- `types/` — Go types not covered by protos: `Verdict` (11 states: PENDING→SKIPPED), contest scoring (`ScoringType`: ICPC/IOI), `ContestLeaderboardEntry`

`Verdict` string representations: "PENDING", "JUDGING", "AC", "WA", "TLE", "MLE", "RE", "CE", "SE", "IE", "SKIPPED".

### Lime (`lime/src`)
- `run.c` (1200+ lines) — main execution logic: user namespaces, overlayfs, capability dropping, cgroup resource enforcement, wall-clock timeout (wall = CPU × 2), 8 MB stdout/stderr cap
- `cgroup.c` — cgroup v2 management
- `api.c` — JSON stdin/stdout API (reads `ExecRequest`, writes `ExecResponse`)

Lime reads an `ExecRequest` JSON from stdin (args, envp, limits, rootfs, bind mounts) and writes an `ExecResponse` (exit code, signal, CPU time, wall time, memory, stdout, stderr).

## Configuration

### jjudge-apiserver
```
SERVER_PORT=8080
JWT_SECRET=<required>
DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_USE_SSL
MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY, MINIO_BUCKET, MINIO_USE_SSL
# Or GCS: GCS_BUCKET, GCS_PROJECT_ID, GCS_CREDENTIALS_FILE
RABBITMQ_URL, RABBITMQ_QUEUE_DURABLE, RABBITMQ_QUEUE_AUTO_DELETE, RABBITMQ_PREFETCH_COUNT
# Or Pub/Sub: PUBSUB_PROJECT_ID, PUBSUB_CREDENTIALS_FILE, PUBSUB_SUBSCRIPTION_SUFFIX
JJUDGE_ADMIN_USER, JJUDGE_ADMIN_PASSWORD
```

### jjudge-worker
```
GRADER_ADDR=localhost:50051
JUDGE_SUBMISSIONS_DIR, JUDGE_WORK_ROOT, JUDGE_OVERLAYFS_DIR, JUDGE_ROOTFS_DIR
JUDGE_MAX_CONCURRENCY=4
RABBITMQ_QUEUE=submissions   # or contest-submissions
LIME_CGROUP_ROOT
ROOTFS_IMG_SRC               # URL to rootfs tarball
# Storage and MQ vars same as apiserver
```

Set `ENV=dev` to load `.env` files automatically.
