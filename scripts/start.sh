#!/usr/bin/env bash
set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Start all backend services via Docker Compose
echo "Starting backend services..."
docker compose -f "$PROJECT_ROOT/docker-compose.yml" up --build -d

# Start frontend dev server
echo "Starting frontend..."
cd "$PROJECT_ROOT/frontend"
npm run dev
