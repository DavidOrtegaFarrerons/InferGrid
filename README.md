# InferGrid

InferGrid is a learning project for building a small self-hosted inference platform in Go. Users submit prompts through gRPC, the backend stores each job in PostgreSQL, RabbitMQ delivers it to a worker and the worker runs the prompt through a local Ollama model such as `llama3.2:1b`.

The initial goal is deliberately small: accept one prompt, process it asynchronously and return the generated result. There is no authentication, chat history, frontend or paid AI API.

## Learning goals

* Design APIs with gRPC and Protocol Buffers
* Process asynchronous jobs with RabbitMQ
* Store durable job state in PostgreSQL
* Integrate Go services with a locally hosted language model
* Understand acknowledgements, retries, idempotency and failure recovery
* Later explore worker scheduling, caching, observability and distributed execution

## MVP

The MVP contains three Go programs:

* An API that accepts and retrieves inference jobs
* A worker that consumes jobs and calls Ollama
* A CLI client used to submit prompts and inspect results

The first version uses one model and one worker. Redis, Elasticsearch, multiple workers, model routing, token streaming and a web interface will be introduced only after the complete basic flow works.
