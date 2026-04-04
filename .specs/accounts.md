# Spec: accounts

## Status: IN_PROGRESS

## Context

O usuário precisa organizar suas finanças pessoais em diferentes "contas" — conta bancária, cartão de crédito, fluxo de caixa, etc. A entidade `Account` modela esses containers financeiros, sem armazenar saldos (serão derivados dos registros de fluxo futuros). Relação N-1 com `users`.

O nome **account** foi escolhido por ser o termo padrão em fintech para representar qualquer container financeiro (bank account, credit card account, cash account).

### Account Types

| Tipo             | Descrição                           |
|------------------|-------------------------------------|
| `bank_account`   | Conta bancária (corrente, poupança) |
| `credit_card`    | Cartão de crédito                   |
| `cash`           | Fluxo de caixa pessoal              |

## Requirements

- [ ] REQ-1: **Criar conta**
  - GIVEN um usuário autenticado
  - WHEN envia POST /accounts com name, type e description (opcional)
  - THEN a conta é criada associada ao user_id do JWT, retorna 201 com id e created_at

- [ ] REQ-2: **Listar contas do usuário**
  - GIVEN um usuário autenticado
  - WHEN envia GET /accounts com filtros opcionais (name, type, active_only) e paginação
  - THEN retorna apenas as contas do próprio usuário (user_id do JWT), com paginação

- [ ] REQ-3: **Buscar conta por ID**
  - GIVEN um usuário autenticado
  - WHEN envia GET /accounts/:id
  - THEN retorna a conta se pertence ao usuário ou se é admin; senão 403

- [ ] REQ-4: **Atualizar conta**
  - GIVEN um usuário autenticado e dono da conta (ou admin)
  - WHEN envia PUT /accounts/:id com name e/ou description (partial update)
  - THEN atualiza os campos fornecidos, retorna 200. Type não pode ser alterado.

- [ ] REQ-5: **Deletar conta (soft delete)**
  - GIVEN um usuário autenticado e dono da conta (ou admin)
  - WHEN envia DELETE /accounts/:id
  - THEN a conta é desativada (active = false), retorna 204

- [ ] REQ-6: **Validação de tipo**
  - GIVEN qualquer operação de criação
  - WHEN o type fornecido não é um dos valores válidos (bank_account, credit_card, cash)
  - THEN retorna 400 com mensagem de erro

- [ ] REQ-7: **Ownership enforcement**
  - GIVEN um usuário autenticado (não-admin, não service-key)
  - WHEN tenta acessar/modificar conta de outro usuário
  - THEN retorna 403 Forbidden

## Design

### Architecture Decisions

- Segue exatamente o padrão do domínio `user`: Clean Architecture com domain → usecases → infrastructure
- Value Object `AccountType` para validação do tipo (similar a `vo.Email`)
- Usa `vo.ID` do domínio user para IDs (UUID v7) — reutiliza o VO existente, sem duplicar
- Ownership check no handler (padrão `isAdminOrOwner` existente), com user_id extraído do JWT context
- Sem cache/singleflight na v1 — simplifica o CRUD inicial
- Permissions RBAC: `account:read`, `account:write`, `account:delete`, `account:update`

### Files to Create

**Domain:**
- `internal/domain/account/entity.go` — Account entity (ID, UserID, Name, Type, Description, Active, timestamps)
- `internal/domain/account/vo/type.go` — AccountType value object (bank_account, credit_card, cash)
- `internal/domain/account/errors.go` — Domain errors (ErrAccountNotFound, ErrInvalidAccountType, ErrForbidden)

**Use Cases:**
- `internal/usecases/account/create.go` — CreateUseCase
- `internal/usecases/account/get.go` — GetUseCase
- `internal/usecases/account/list.go` — ListUseCase
- `internal/usecases/account/update.go` — UpdateUseCase
- `internal/usecases/account/delete.go` — DeleteUseCase
- `internal/usecases/account/observability.go` — injectLogContext helper
- `internal/usecases/account/interfaces/repository.go` — Repository interface
- `internal/usecases/account/dto/create.go` — Create DTOs
- `internal/usecases/account/dto/get.go` — Get DTOs
- `internal/usecases/account/dto/list.go` — List DTOs
- `internal/usecases/account/dto/update.go` — Update DTOs
- `internal/usecases/account/dto/delete.go` — Delete DTOs

**Infrastructure:**
- `internal/infrastructure/db/postgres/repository/account.go` — PostgreSQL repository
- `internal/infrastructure/web/handler/account.go` — Gin HTTP handler
- `internal/infrastructure/web/router/account.go` — Route registration

**Migration:**
- `internal/infrastructure/db/postgres/migration/<timestamp>_create_accounts_table.sql`

**Tests:**
- `internal/usecases/account/create_test.go`
- `internal/usecases/account/get_test.go`
- `internal/usecases/account/list_test.go`
- `internal/usecases/account/update_test.go`
- `internal/usecases/account/delete_test.go`
- `internal/usecases/account/mocks_test.go`
- `internal/domain/account/entity_test.go`

### Files to Modify

- `internal/infrastructure/web/handler/error.go` — Adicionar mappings de erros do domínio account
- `internal/infrastructure/web/router/router.go` — Adicionar `AccountHandler` em Dependencies + registrar rotas
- `cmd/api/server.go` — Wiring do account domain em `buildDependencies()`
- Migration RBAC existente ou nova migration — Seed permissions `account:read`, `account:write`, `account:delete`, `account:update`

### Dependencies

Nenhuma dependência externa nova. Usa os mesmos packages: `sqlx`, `gin`, `otel`, `vo.ID` (reutilizado do user domain).

## Tasks

- [x] TASK-1: **Migration — criar tabela `accounts` + seed permissions**
  Criar migration SQL com tabela `accounts` (id UUID PK, user_id UUID FK→users, name VARCHAR(255), type VARCHAR(50), description TEXT, active BOOLEAN, created_at/updated_at TIMESTAMPTZ). Índices em user_id e type. Seed permissions account:read, account:write, account:delete, account:update e associar ao role admin. Incluir Down migration reversível.
  Ref: `internal/infrastructure/db/postgres/migration/20260402001_create_rbac_tables.sql`

- [x] TASK-2: **Domain — entity, value objects, errors**
  Criar `internal/domain/account/entity.go` (Account struct + NewAccount factory + métodos Deactivate, UpdateName, UpdateDescription), `vo/type.go` (AccountType com validação bank_account|credit_card|cash), `errors.go` (ErrAccountNotFound, ErrInvalidAccountType, ErrForbidden). Reutilizar `user/vo.ID` para IDs.
  Ref: `internal/domain/user/entity.go`, `internal/domain/user/vo/`

- [x] TASK-3: **Domain tests — entity + value object**
  Criar `internal/domain/account/entity_test.go` com testes para NewAccount, Deactivate, UpdateName, UpdateDescription. Testar AccountType validation (válidos e inválidos).
  Ref: `internal/domain/user/entity_test.go`

- [x] TASK-4: **Use case interfaces + DTOs**
  Criar `internal/usecases/account/interfaces/repository.go` (Create, FindByID, List, Update, Delete). Criar DTOs: create (name, type, description? → id, created_at), get (id → full account), list (page, limit, name?, type?, active_only? → paginated), update (id, name?, description? → full account), delete (id → void). Criar `observability.go`.
  Ref: `internal/usecases/user/interfaces/`, `internal/usecases/user/dto/`

- [x] TASK-5: **Use cases — Create + Get**
  Implementar CreateUseCase (valida type VO, cria entity, persiste) e GetUseCase (busca por ID). Ambos com OTel spans e logging.
  Ref: `internal/usecases/user/create.go`, `internal/usecases/user/get.go`

- [x] TASK-6: **Use cases — List + Update + Delete**
  Implementar ListUseCase (filtros + paginação, scoped por user_id), UpdateUseCase (partial update de name/description, invalida nada por enquanto), DeleteUseCase (soft delete). Todos com OTel spans.
  Ref: `internal/usecases/user/list.go`, `internal/usecases/user/update.go`, `internal/usecases/user/delete.go`

- [x] TASK-7: **Use case tests + mocks**
  Criar `mocks_test.go` (MockRepository com testify/mock). Testes para todos os 5 use cases: happy path + error paths (not found, invalid type, repo error).
  Ref: `internal/usecases/user/mocks_test.go`, `internal/usecases/user/create_test.go`

- [x] TASK-8: **Repository — PostgreSQL implementation**
  Criar `internal/infrastructure/db/postgres/repository/account.go` com accountDB model, conversões domain↔DB, queries CRUD. List com dynamic WHERE (user_id obrigatório + filtros opcionais), read-only transaction para paginação.
  Ref: `internal/infrastructure/db/postgres/repository/user.go`

- [x] TASK-9: **Handler — HTTP handlers + error mapping**
  Criar `internal/infrastructure/web/handler/account.go` (AccountHandler struct com 5 use cases). Cada handler: bind request, extrair user_id do JWT context, ownership check (isAdminOrOwner), executar use case, responder. Adicionar erros do account domain em `error.go`.
  Ref: `internal/infrastructure/web/handler/user.go`, `internal/infrastructure/web/handler/error.go`

- [x] TASK-10: **Router + DI wiring**
  Criar `internal/infrastructure/web/router/account.go` (RegisterAccountRoutes com permissions account:read/write/delete). Adicionar AccountHandler em router.Dependencies. Wiring em `cmd/api/server.go:buildDependencies()`: accountRepo → use cases → handler → dependencies.
  Ref: `internal/infrastructure/web/router/user.go`, `internal/infrastructure/web/router/router.go`

## Validation Criteria

- [ ] `go build ./...` passa
- [ ] `make lint` passa
- [ ] `make test-unit` passa (todos os testes de domain + use cases)
- [ ] Migration aplica e reverte sem erros
- [ ] Swagger docs gerados sem erros (`swag init ...`)

## Execution Log

<!-- Ralph Loop appends here automatically — do not edit manually -->

### Iteration 1 — TASK-1 (2026-04-03 10:00)

Created migration `20260403001_create_accounts_table.sql` with accounts table (id, user_id FK, name, type, description, active, timestamps), indexes on user_id/type/composite, seeded 4 permissions (account:read/write/update/delete) and assigned to admin+user roles. Reversible Down included.

### Iteration 2 — TASK-2 (2026-04-03 10:05)

Created domain layer: `entity.go` (Account struct with NewAccount factory, Deactivate/UpdateName/UpdateDescription methods), `vo/type.go` (AccountType VO with validation for bank_account|credit_card|cash, driver.Valuer/sql.Scanner), `vo/errors.go`, `errors.go` (ErrAccountNotFound, ErrForbidden). Reuses `user/vo.ID` for IDs.

### Iteration 3 — TASK-3 (2026-04-03 10:10)

Created `entity_test.go` with 6 tests: NewAccount, Deactivate, UpdateName, UpdateDescription, AccountType valid (table-driven, 3 types), AccountType invalid (table-driven, 4 cases). All pass.

### Iteration 4 — TASK-4 (2026-04-03 10:15)

Created use case layer scaffolding: `interfaces/repository.go` (5 methods), DTOs for all 5 operations (create, get, list, update, delete), `observability.go`, and domain `filter.go` (ListFilter + ListResult). All following existing user/role patterns.

### Iteration 5 — TASK-5 (2026-04-03 10:20)

Created `create.go` (validates UserID + AccountType VOs, creates entity, persists) and `get.go` (validates ID, finds by ID via repo). Both with OTel spans and structured logging.

### Iteration 6 — TASK-6 (2026-04-03 10:25)

Created `list.go` (user_id-scoped filter + pagination), `update.go` (partial update name/description via FindByID + domain methods), `delete.go` (soft delete). All with OTel spans and structured logging.

### Iteration 7 — TASK-7 (2026-04-03 10:30)

Created `mocks_test.go` (MockRepository) and 5 test files with 21 tests total: create (4), get (4), list (4), update (5), delete (4). Covers happy paths, validation errors, not found, and repository errors. All pass.

### Iteration 8 — TASK-8 (2026-04-03 10:35)

Created `repository/account.go` with accountDB model, toAccount/fromDomainAccount conversions, full CRUD: Create (named exec), FindByID (ErrNoRows→ErrAccountNotFound), List (dynamic WHERE with user_id scoping, read-only tx for pagination), Update (tx with rowsAffected check), Delete (soft delete). Added Offset() method to domain filter.

### Iteration 9 — TASK-9 (2026-04-03 10:40)

Created `handler/account.go` with AccountHandler (5 use cases, no metrics). Ownership check fetches account then compares UserID via isAdminOrOwner. Create/List extract user_id from JWT context. Delete returns 204. Added 3 account domain error mappings to `error.go` (ErrInvalidAccountType, ErrAccountNotFound, ErrForbidden).

### Iteration 10 — TASK-10 (2026-04-03 10:45)

Created `router/account.go` (RegisterAccountRoutes with account:read/write/update/delete permissions). Added AccountHandler to router.Dependencies. Wired account domain in `server.go:buildDependencies()`: repo → 5 use cases → handler → dependencies. Routes registered with JWT auth when enabled.
