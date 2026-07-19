# ADR 0001: Orchestrated saga over choreographed saga

## Status
Accepted

## Context
The mechanic-shop POS is being split into 4 services (OS Service, Billing
Service, Production Service, Saga Orchestrator) that must coordinate
multi-step, cross-service business transactions (e.g. starting execution of
an OS, billing on completion) without distributed ACID transactions. Two
standard patterns exist: choreography (each service reacts to events from
its peers and emits its own, with no central coordinator) and orchestration
(a dedicated Saga Orchestrator service owns the transaction's steps and
sends explicit commands to participants).

## Decision
Use an orchestrated saga. A dedicated `pos-saga-orchestrator` service issues
commands (e.g. `StartExecutionCommand`) to this service and the others, and
this service reacts by performing its local transactional work and emitting
domain events (`ExecutionStarted`, `ExecutionCompleted`, `ExecutionFailed`)
that the orchestrator consumes to decide the next step or trigger
compensation.

## Consequences
- This service only needs to know its own command-in/event-out contract; it
  has no knowledge of the overall saga graph or of its sibling services.
- The orchestrator is the single place that encodes saga step ordering and
  compensation logic, which is easier to reason about and test in a course
  setting than compensation logic scattered across 4 independently-built
  services.
- Adds a runtime dependency on the orchestrator being available and correct;
  this service alone cannot recover a stuck saga.
- All inter-service messages still flow over the shared `pos.events` /
  `pos.retry` RabbitMQ topology, so the two patterns are not mutually
  exclusive at the transport level -- orchestration is a decision about who
  decides "what happens next", not about the messaging infrastructure.
