# JJudge — single-node Makefile
#
# Usage:
#   make dev      — build images and start all services in the foreground (logs streamed)
#   make deploy   — build images and start all services in the background
#   make stop     — stop and remove containers (volumes preserved)
#   make destroy  — stop and remove containers AND volumes (wipes database / MinIO)
#   make restart  — restart all services without rebuilding
#   make logs     — tail logs from all running containers
#   make ps       — show container status
#   make build    — build (or rebuild) images without starting
#   make setup    — run the one-time Ubuntu host setup script

COMPOSE      := docker compose
COMPOSE_FILE := docker-compose.yml
PROJECT_ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

.PHONY: dev deploy stop destroy restart logs ps build setup \
        cgroup frontend-install

# ── Dev ───────────────────────────────────────────────────────────────────────
# Builds images, starts services, and streams logs to the terminal.
# Ctrl-C stops all containers.
dev: cgroup
	$(COMPOSE) -f $(COMPOSE_FILE) up --build

# ── Deploy (production / background) ─────────────────────────────────────────
# Builds images and runs everything detached.
# Useful for a live server; combine with 'make logs' to watch output.
deploy: cgroup
	BUILD_TARGET=prod $(COMPOSE) -f $(COMPOSE_FILE) up --build -d
	@echo ""
	@echo "Services started in background. Useful commands:"
	@echo "  make logs    — tail all logs"
	@echo "  make ps      — show status"
	@echo "  make stop    — shut down"

# ── Stop ──────────────────────────────────────────────────────────────────────
stop:
	$(COMPOSE) -f $(COMPOSE_FILE) down

# ── Destroy (wipe data volumes) ───────────────────────────────────────────────
destroy:
	@echo "WARNING: This will delete all database and MinIO data."
	@read -p "Are you sure? [y/N] " ans && [ "$$ans" = "y" ]
	$(COMPOSE) -f $(COMPOSE_FILE) down -v

# ── Restart ───────────────────────────────────────────────────────────────────
restart:
	$(COMPOSE) -f $(COMPOSE_FILE) restart

# ── Logs ──────────────────────────────────────────────────────────────────────
logs:
	$(COMPOSE) -f $(COMPOSE_FILE) logs -f

# ── Status ────────────────────────────────────────────────────────────────────
ps:
	$(COMPOSE) -f $(COMPOSE_FILE) ps

# ── Build images only ─────────────────────────────────────────────────────────
build:
	$(COMPOSE) -f $(COMPOSE_FILE) build

# ── Ensure lime.slice cgroup exists (required before worker starts) ───────────
# Runs the init script directly in case the systemd service hasn't fired yet.
cgroup:
	@if [ -f /usr/local/bin/jjudge-cgroup-init.sh ]; then \
	    sudo /usr/local/bin/jjudge-cgroup-init.sh; \
	else \
	    echo "jjudge-cgroup-init.sh not found — run 'make setup' first."; \
	    exit 1; \
	fi

# ── First-time host setup ─────────────────────────────────────────────────────
setup:
	sudo bash $(PROJECT_ROOT)scripts/setup.sh

# ── Install frontend npm deps (for local development outside Docker) ──────────
frontend-install:
	cd $(PROJECT_ROOT)frontend && npm install
