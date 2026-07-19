set shell := ["bash", "-uc"]
set dotenv-load := true

api_cmd := env_var_or_default("API_CMD", "./cmd/api")
worker_cmd := env_var_or_default("WORKER_CMD", "./cmd/worker")
relay_cmd := env_var_or_default("RELAY_CMD", "./cmd/relay")
client_cmd := env_var_or_default("CLIENT_CMD", "./cmd/client")

default: help

help:
    @printf "%-18s %s\n" \
        "just setup" "Create .env, download dependencies and start infrastructure" \
        "just infrastructure" "Start Docker Compose services" \
        "just down" "Stop Docker Compose services" \
        "just logs" "Follow Docker Compose logs" \
        "just api" "Run the API" \
        "just worker" "Run the worker" \
        "just relay" "Run the outbox relay" \
        "just client" "Run the client" \
        "just dev" "Run API, relay and worker together" \
        "just test" "Run unit tests" \
        "just test-integration" "Run unit and integration tests (needs Docker)" \
        "just check" "Run tests, vet and build"

setup:
    test -f .env || cp .env.example .env
    go mod download
    git config core.hooksPath .githooks
    docker compose up -d

infrastructure:
    docker compose up -d

down:
    docker compose down

logs:
    docker compose logs -f

api: check-api-env
    go run {{api_cmd}}

worker: check-worker-env
    go run {{worker_cmd}}

relay: check-relay-env
    go run {{relay_cmd}}

client *ARGS: check-client-env
    go run ./cmd/client {{ARGS}}

submit PROMPT:
    @test -n "{{PROMPT}}" || (echo "PROMPT is required"; exit 1)
    go run ./cmd/client submit "{{PROMPT}}"

get JOB_ID:
    @test -n "{{JOB_ID}}" || (echo "JOB_ID is required"; exit 1)
    go run ./cmd/client get "{{JOB_ID}}"

dev: check-api-env check-relay-env check-worker-env
    #!/usr/bin/env bash
    set -euo pipefail
    go run {{api_cmd}} & api_pid=$!
    go run {{relay_cmd}} & relay_pid=$!
    go run {{worker_cmd}} & worker_pid=$!
    trap 'kill $api_pid $relay_pid $worker_pid 2>/dev/null || true' EXIT INT TERM
    wait

test:
    go test ./...

test-integration:
    go test -tags integration ./...

check:
    go test ./...
    go vet ./...
    go build ./...

check-api-env:
    #!/usr/bin/env bash
    for key in DATABASE_DSN MIGRATIONS_PATH AMQP_URL GRPC_LISTEN_ADDRESS; do
        value="${!key}"
        if [ -z "$value" ]; then echo "$key is required in .env"; exit 1; fi
    done

check-worker-env:
    #!/usr/bin/env bash
    for key in DATABASE_DSN MIGRATIONS_PATH AMQP_URL; do
        value="${!key}"
        if [ -z "$value" ]; then echo "$key is required in .env"; exit 1; fi
    done

check-relay-env:
    #!/usr/bin/env bash
    for key in DATABASE_DSN AMQP_URL GRPC_LISTEN_ADDRESS; do
        value="${!key}"
        if [ -z "$value" ]; then echo "$key is required in .env"; exit 1; fi
    done

check-client-env:
    #!/usr/bin/env bash
    for key in GRPC_LISTEN_ADDRESS; do
        value="${!key}"
        if [ -z "$value" ]; then echo "$key is required in .env"; exit 1; fi
    done
