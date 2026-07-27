# ADR 0001: Saga orquestrada em vez de saga coreografada

## Status
Aceito

## Contexto
O PDV da oficina mecânica está sendo dividido em 4 serviços (OS Service,
Billing Service, Production Service, Saga Orchestrator) que precisam
coordenar transações de negócio multi-etapa e entre serviços (por exemplo,
iniciar a execução de uma OS, faturar ao concluir) sem transações ACID
distribuídas. Existem dois padrões consolidados: coreografia (cada serviço
reage a eventos dos seus pares e emite os próprios, sem coordenador central)
e orquestração (um serviço dedicado Saga Orchestrator é dono das etapas da
transação e envia comandos explícitos aos participantes).

## Decisão
Usar uma saga orquestrada. Um serviço dedicado `pos-saga-orchestrator` emite
comandos (por exemplo, `StartExecutionCommand`) para este serviço e para os
demais, e este serviço reage executando seu trabalho transacional local e
emitindo eventos de domínio (`ExecutionStarted`, `ExecutionCompleted`,
`ExecutionFailed`) que o orquestrador consome para decidir o próximo passo
ou disparar compensação.

## Consequências
- Este serviço só precisa conhecer seu próprio contrato de comando-entrada/
  evento-saída; ele não tem conhecimento do grafo geral da saga nem dos
  outros serviços.
- O orquestrador é o único lugar que concentra a ordenação das etapas da
  saga e a lógica de compensação, o que é mais fácil de raciocinar e testar
  em um contexto de curso do que lógica de compensação espalhada por 4
  serviços construídos de forma independente.
- Adiciona uma dependência em tempo de execução da disponibilidade e
  correção do orquestrador; este serviço sozinho não consegue recuperar uma
  saga travada.
- Todas as mensagens entre serviços continuam fluindo pela mesma topologia
  compartilhada de RabbitMQ `pos.events` / `pos.retry`, então os dois
  padrões não são mutuamente exclusivos no nível de transporte --
  orquestração é uma decisão sobre quem decide "o que acontece a seguir",
  não sobre a infraestrutura de mensageria.
