# SDD + Ralph Loop — Desenvolvimento Orientado a Especificação

## Visão Geral

O template inclui um fluxo de desenvolvimento integrado baseado em dois conceitos:

- **SDD (Specification-Driven Development)** — Metodologia onde especificações estruturadas guiam a implementação, em vez de codar diretamente
- **Ralph Loop** — Técnica de execução autônoma onde o agente de IA executa tasks uma a uma, com progresso rastreado no disco

Juntos, formam um workflow completo: **especificar → aprovar → executar → validar**.

## Quick Start

```text
# 1. Gerar uma especificação
/spec "Add audit logging to user write operations"

# 2. Revisar e aprovar a spec gerada em .specs/

# 3. Executar autonomamente
/ralph-loop .specs/user-audit-log.md

# 4. (Opcional) Revisar implementação contra spec
/spec-review .specs/user-audit-log.md
```

## Quando Usar

| Cenário | Recomendação |
| --- | --- |
| Feature nova com 3+ tasks | `/spec` + `/ralph-loop` |
| Bug fix simples | `/fix-issue` (workflow existente) |
| Novo endpoint isolado | `/new-endpoint` ou `/spec` (depende da complexidade) |
| Refactor cross-cutting | `/spec` + execução manual (task por task) |
| Tarefa trivial (1-2 arquivos) | Execução direta, sem spec |

**Princípio: profundidade proporcional** — a spec deve ser tão detalhada quanto a complexidade da task exige.

## Estrutura da Spec

As specs ficam em `.specs/` e seguem o template em `.specs/TEMPLATE.md`:

```markdown
# Spec: user-audit-log

## Status: DRAFT

## Context
Precisamos de audit logging para todas as operações de escrita em user
para compliance com requisitos de segurança.

## Requirements
- [ ] REQ-1: GIVEN um POST /api/v1/users WHEN a operação é bem-sucedida
      THEN um evento de auditoria é registrado com user_id, action, timestamp
- [ ] REQ-2: ...

## Design
### Architecture Decisions
Criar um novo domínio `audit` com entity AuditEvent.
Usar o padrão observer para capturar eventos nos use cases existentes.

### Files to Create
- internal/domain/audit/entity.go
- internal/usecases/audit/create.go
- ...

### Files to Modify
- cmd/api/server.go (DI wiring)

## Tasks
- [ ] TASK-1: Criar entity AuditEvent com campos (ID, UserID, Action, ...)
- [ ] TASK-2: Criar use case CreateAuditEvent com interface de repository
- [ ] TASK-3: Implementar repository PostgreSQL para AuditEvent
- [ ] TASK-4: Integrar audit logging nos use cases de user (create, update, delete)
- [ ] TASK-5: Adicionar migration para tabela audit_events
- [ ] TASK-6: Escrever testes unitários para audit domain e use cases

## Validation Criteria
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Eventos de auditoria são criados após POST/PUT/DELETE em /api/v1/users

## Execution Log
<!-- Ralph Loop preenche automaticamente -->
```

### Regras da Spec

- **Status** é uma state machine: `DRAFT` → `APPROVED` → `IN_PROGRESS` → `DONE` | `FAILED`
- **Requirements** usam formato GIVEN/WHEN/THEN para acceptance criteria não ambíguos
- **Tasks** são agnósticas de arquitetura — ordenadas logicamente para a feature, sem impor camadas
- Items incertos devem ser marcados `[NEEDS CLARIFICATION]`
- Cada task deve ser verificável independentemente (`go build ./...` deve passar após cada uma)

## Como o Ralph Loop Funciona

### Mecanismo

```text
Claude executa 1 task → marca [x] → tenta parar
  ↓
ralph-loop.sh (Stop hook):
  - .active.md existe? → conta tasks restantes
  - Tasks restantes > 0? → exit 2 (continua na mesma sessão)
  - Tasks restantes = 0? → exit 0 (passa para validação final)
  ↓
stop-validate.sh:
  - Durante ralph loop: skip (detecta .active.md)
  - Após ralph loop: validação completa (build + lint + tests)
```

### Ciclo de Vida

1. **Startup**: `/ralph-loop` lê a spec, cria `.specs/<name>.active.md`, identifica o primeiro task
2. **Iteração**: executa 1 task, marca `[x]`, loga no Execution Log, para
3. **Continuação**: Stop hook detecta tasks restantes, re-injeta contexto via stderr, Claude continua
4. **Finalização**: todos os tasks `[x]`, hook retorna exit 0, validação completa roda
5. **Resultado**: spec marcada como `DONE` (ou `FAILED` se validação falhar)

### Limites e Safety

- **Máximo 30 iterações** por sessão (counter em `/tmp/ralph-loop-<session>`)
- **Build check** a cada iteração (detecta erros de compilação cedo)
- **Contexto acumulado**: exit code 2 continua na mesma sessão (não reinicia). Para features com muitas tasks (15+), divida em specs menores
- **Emergency stop**: `rm .specs/*.active.md`
- **Resume**: se interrompido, `/ralph-loop` retoma do último task incompleto

## Integração com Workflows Existentes

O SDD + Ralph Loop se integra sem quebrar nada:

| Workflow | Integração |
| --- | --- |
| `/new-endpoint` | Pode ser usado como referência para tasks na spec |
| `/fix-issue` | Para bugs, continue usando `/fix-issue` — mais direto |
| `/validate` | Roda automaticamente no final do ralph-loop via Stop hook |
| `/review`, `/full-review-team` | Use após ralph-loop para review mais profundo |
| `/spec-review` | Review específico contra os requirements da spec |
| Lefthook (pre-commit) | Continua rodando normalmente nos commits |
| `lint-go-file.sh` (PostToolUse) | Roda a cada edit durante o ralph-loop |

## Referências

- [Specification-Driven Development (Thoughtworks/Martin Fowler)](https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html)
- [Ralph Wiggum Technique (Geoffrey Huntley)](https://ghuntley.com/loop/)
- [Claude Code Hooks — Stop hook exit codes](https://docs.anthropic.com/en/docs/claude-code/hooks)
