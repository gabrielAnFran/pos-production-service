# Production Service

Execution/repair queue microservice for the mechanic-shop POS system. Owns
the lifecycle of an OS's physical execution (diagnosing -> repairing ->
completed/failed) once a saga has authorized work to start.

## Why MongoDB

This is the one NoSQL service in the split, satisfying the project's NoSQL
requirement. An execution record (status, technician, notes, timestamps) plus
its append-only repair history is naturally document-shaped and doesn't need
relational joins, so MongoDB with collection-per-concern
(`execution_queue`, `repair_history`, `outbox`, `processed_events`) is a
reasonable fit.

## Architecture

- **cmd/server**: Gin HTTP API (`GET/PATCH /api/v1/executions/:os_id`,
  `/healthz`, `/readyz`).
- **cmd/worker**: consumes `StartExecutionCommand` from RabbitMQ and runs the
  outbox dispatcher goroutine in the same process.
- **Outbox pattern with polling** (not change streams): domain writes and
  the corresponding outbox event are written in a single MongoDB
  multi-document transaction (`session.WithTransaction`), then a dispatcher
  goroutine polls the `outbox` collection every
  `PRODUCTION_DISPATCH_INTERVAL_MS` (default 500ms) for unpublished rows,
  publishes them to RabbitMQ, and marks them published. This avoids the
  added operational complexity of change streams while still getting
  at-least-once, transactionally-consistent event publication.

## Single-node replica set requirement

MongoDB only supports multi-document transactions against a replica set
(even a single-node one). `PRODUCTION_MONGO_URI` must point at a Mongo
reachable as a replica set (e.g. `mongod --replSet rs0` plus one
`rs.initiate()` call on first boot). This is expected to be handled by
`pos-saga-orchestrator/deploy/local/docker-compose.yml`; this repo does not
own that compose file.

## Running locally

As part of the full stack:

```
# from pos-saga-orchestrator/deploy/local/
docker-compose up
```

Standalone (requires a reachable replica-set Mongo and RabbitMQ):

```
export PRODUCTION_MONGO_URI="mongodb://localhost:27017/production?replicaSet=rs0"
export PRODUCTION_AMQP_URL="amqp://guest:guest@localhost:5672/"
go run ./cmd/server &
go run ./cmd/worker
```

## Environment variables

| Variable                          | Default                                                              |
|------------------------------------|-----------------------------------------------------------------------|
| `PRODUCTION_PORT`                  | `8083`                                                                |
| `PRODUCTION_MONGO_URI`             | `mongodb://production-mongo:27017/production?replicaSet=rs0`         |
| `PRODUCTION_AMQP_URL`              | `amqp://guest:guest@localhost:5672/`                                 |
| `PRODUCTION_DISPATCH_INTERVAL_MS`  | `500`                                                                 |

## Testing

- Unit tests (no external services needed): `go test ./...`.
- Integration test (real, single-node-replica-set Mongo via
  testcontainers-go): `make test-integration` (build tag `integration`).
  Not run in CI (see `.github/workflows/ci.yml` for why); run it locally or
  against the docker-compose stack.
- Coverage at last run: ~22.5% total statements (`go tool cover -func`).
  Concentrated in domain/entities (100%) and use cases (78.3%); the Mongo
  repository and AMQP transport are covered by the integration test rather
  than unit tests, per the challenge's testing guidance.

## Saga participation

- **Consumes**: `StartExecutionCommand` (`os_id`, `budget_id`) -- creates an
  execution in `DIAGNOSING` status.
- **Produces**:
  - `ExecutionStarted` (`os_id`, `started_at`, `technician_id`)
  - `ExecutionCompleted` (`os_id`, `completed_at`, `repair_notes`)
  - `ExecutionFailed` (`os_id`, `reason`)

Status transitions `DIAGNOSING -> REPAIRING -> {COMPLETED, FAILED}` (and
`DIAGNOSING -> FAILED`) are driven via `PATCH /api/v1/executions/:os_id`.
The `REPAIRING` transition is internal to this service and not in the event
catalog, so no event is emitted for it -- only `repair_history` is updated.

See `docs/adr/0001-orchestrated-saga.md` for why the saga is orchestrated
rather than choreographed, and `docs/openapi.yaml` for the full REST
contract.
