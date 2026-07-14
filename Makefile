SHELL := /bin/bash
.DEFAULT_GOAL := help

API_CMD ?= ./cmd/inference/api
WORKER_CMD ?= ./cmd/inference/worker
CLIENT_CMD ?= ./cmd/client

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: help setup infrastructure down logs api worker client dev check check-api-env check-worker-env check-client-env

help:
	@printf "%-18s %s\n" \
		"make setup" "Create .env, download dependencies and start infrastructure" \
		"make infrastructure" "Start Docker Compose services" \
		"make down" "Stop Docker Compose services" \
		"make logs" "Follow Docker Compose logs" \
		"make api" "Run the API" \
		"make worker" "Run the worker" \
		"make client" "Run the client" \
		"make dev" "Run API and worker together" \
		"make check" "Run tests, vet and build"

setup:
	@test -f .env || cp .env.example .env
	go mod download
	docker compose up -d

infrastructure:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

api: check-api-env
	go run $(API_CMD)

worker: check-worker-env
	go run $(WORKER_CMD)

client: check-client-env
	go run ./cmd/client $(ARGS)

submit:
	@test -n "$(PROMPT)" || (echo "PROMPT is required"; exit 1)
	go run ./cmd/client submit "$(PROMPT)"

get:
	@test -n "$(JOB_ID)" || (echo "JOB_ID is required"; exit 1)
	go run ./cmd/client get "$(JOB_ID)"

dev: check-api-env check-worker-env
	@set -euo pipefail; \
	go run $(API_CMD) & api_pid=$$!; \
	go run $(WORKER_CMD) & worker_pid=$$!; \
	trap 'kill $$api_pid $$worker_pid 2>/dev/null || true' EXIT INT TERM; \
	wait

check:
	go test ./...
	go vet ./...
	go build ./...

check-api-env:
	@for key in DATABASE_DSN MIGRATIONS_PATH AMQP_URL GRPC_LISTEN_ADDRESS; do \
		value="$${!key}"; \
		if [ -z "$$value" ]; then echo "$$key is required in .env"; exit 1; fi; \
	done

check-worker-env:
	@for key in DATABASE_DSN MIGRATIONS_PATH AMQP_URL OLLAMA_URL OLLAMA_MODEL; do \
		value="$${!key}"; \
		if [ -z "$$value" ]; then echo "$$key is required in .env"; exit 1; fi; \
	done

check-client-env:
	@for key in GRPC_LISTEN_ADDRESS; do \
		value="$${!key}"; \
		if [ -z "$$value" ]; then echo "$$key is required in .env"; exit 1; fi; \
	done