# Production Service

Microsserviço de fila de execução/reparo do sistema de PDV para oficina
mecânica. É responsável pelo ciclo de vida da execução física de uma OS
(diagnosticando -> reparando -> concluída/falhou) a partir do momento em que
uma saga autoriza o início do trabalho.

## Por que MongoDB

Este é o único serviço NoSQL da divisão, atendendo ao requisito de NoSQL do
projeto. Um registro de execução (status, técnico, notas, timestamps) e seu
histórico de reparos append-only têm um formato naturalmente orientado a
documentos e não precisam de joins relacionais, então o MongoDB com uma
coleção por responsabilidade (`execution_queue`, `repair_history`, `outbox`,
`processed_events`) é uma escolha razoável.

## Arquitetura

- **cmd/server**: API HTTP em Gin (`GET/PATCH /api/v1/executions/:os_id`,
  `/healthz`, `/readyz`).
- **cmd/worker**: consome `StartExecutionCommand` do RabbitMQ e roda a
  goroutine do dispatcher da outbox no mesmo processo.
- **Padrão outbox com polling** (e não change streams): as escritas de
  domínio e o evento de outbox correspondente são gravados em uma única
  transação multi-documento do MongoDB (`session.WithTransaction`), e então
  uma goroutine dispatcher faz polling da coleção `outbox` a cada
  `PRODUCTION_DISPATCH_INTERVAL_MS` (padrão 500ms) em busca de linhas ainda
  não publicadas, publica no RabbitMQ e as marca como publicadas. Isso evita
  a complexidade operacional extra de change streams, mantendo publicação de
  eventos com garantia at-least-once e consistência transacional.

## Requisito de replica set single-node

O MongoDB só suporta transações multi-documento contra um replica set
(mesmo que seja de um único nó). `PRODUCTION_MONGO_URI` precisa apontar para
um Mongo acessível como replica set (por exemplo, `mongod --replSet rs0`
mais uma chamada de `rs.initiate()` no primeiro boot). Isso é esperado ser
tratado por `pos-saga-orchestrator/deploy/local/docker-compose.yml`; este
repositório não é dono desse arquivo compose.

## Rodando localmente

Como parte da stack completa:

```
# a partir de pos-saga-orchestrator/deploy/local/
docker-compose up
```

Standalone (requer um Mongo replica-set acessível e RabbitMQ):

```
export PRODUCTION_MONGO_URI="mongodb://localhost:27017/production?replicaSet=rs0"
export PRODUCTION_AMQP_URL="amqp://guest:guest@localhost:5672/"
go run ./cmd/server &
go run ./cmd/worker
```

## Variáveis de ambiente

| Variável                          | Padrão                                                                |
|------------------------------------|-----------------------------------------------------------------------|
| `PRODUCTION_PORT`                  | `8083`                                                                |
| `PRODUCTION_MONGO_URI`             | `mongodb://production-mongo:27017/production?replicaSet=rs0`         |
| `PRODUCTION_AMQP_URL`              | `amqp://guest:guest@localhost:5672/`                                 |
| `PRODUCTION_DISPATCH_INTERVAL_MS`  | `500`                                                                 |

## Testes

- Testes unitários (sem necessidade de serviços externos): `go test ./...`.
- Testes de integração (Mongo replica-set single-node real + RabbitMQ real
  via testcontainers-go, requer Docker): `make test-integration` (build tag
  `integration`). Também roda no CI como parte do job `test` -- ambas as
  suítes provisionam seus próprios containers, então nenhum bloco
  `services:` ou Mongo/RabbitMQ iniciado manualmente é necessário.
- Cobertura na última execução (`go test -tags=integration ./... -coverpkg=./...`,
  combinando unitários + integração): **67,4%** do total de statements.
  Note que `go test ./... -coverprofile=...` simples (sem `-coverpkg=./...`)
  subestima esse número, já que `db` e `messaging` só são exercitados pelo
  pacote externo `tests/integration`.
  - `internal/domain/entities`, `internal/infrastructure/config`,
    `internal/presentation/dto`, `internal/presentation/middleware`: 100%
  - `internal/presentation/handlers`: 97,6%
  - `internal/infrastructure/db`: 81,8% (repositório Mongo, escritas
    transacionais na outbox, todas as transições de status, idempotência)
  - `internal/application/usecases`: 78,3%
  - `internal/infrastructure/messaging`: 68,6% (roundtrip de publish/consume,
    retry -> DLQ após `MaxRetries`, dispatcher da outbox)
  - `cmd/server`, `cmd/worker`: 0% (apenas wiring do main(), sem teste
    unitário)

## Participação na saga

- **Consome**: `StartExecutionCommand` (`os_id`, `budget_id`) -- cria uma
  execução em status `DIAGNOSING`.
- **Produz**:
  - `ExecutionStarted` (`os_id`, `started_at`, `technician_id`)
  - `ExecutionCompleted` (`os_id`, `completed_at`, `repair_notes`)
  - `ExecutionFailed` (`os_id`, `reason`)

As transições de status `DIAGNOSING -> REPAIRING -> {COMPLETED, FAILED}` (e
`DIAGNOSING -> FAILED`) são feitas via `PATCH /api/v1/executions/:os_id`. A
transição `REPAIRING` é interna a este serviço e não faz parte do catálogo
de eventos, então nenhum evento é emitido para ela -- apenas o
`repair_history` é atualizado.

Veja `docs/adr/0001-orchestrated-saga.md` para entender por que a saga é
orquestrada em vez de coreografada, e `docs/openapi.yaml` para o contrato
REST completo.
