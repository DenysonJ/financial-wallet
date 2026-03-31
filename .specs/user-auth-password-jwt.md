# Spec: user-auth-password-jwt

## Status: IN_PROGRESS

## Context

O sistema atualmente gerencia usuários (CRUD) mas não possui autenticação própria. A autenticação atual é service-to-service via Service Key (ADR-005). Este feature adiciona autenticação de usuário final com senha (bcrypt) e JWT, permitindo que clientes façam login e usem o token para acessar endpoints protegidos.

Fluxo de negócio:
1. Usuário é criado (CRUD existente) — sem senha inicialmente
2. Usuário cadastra senha (set-password) — senha + confirmação
3. Usuário faz login (email + senha) → recebe access token JWT + refresh token
4. Usuário usa o access token no header `Authorization: Bearer <token>` para acessar endpoints protegidos
5. Usuário pode alterar senha (change-password) — senha atual + nova + confirmação
6. Refresh token permite renovar o access token sem re-login

## Requirements

- [ ] REQ-1: Cadastro de senha
  - GIVEN um usuário existente sem senha cadastrada
  - WHEN o endpoint `POST /users/password` é chamado com `password`, `password_confirmation` e o ID do usuário
  - THEN a senha é salva como hash bcrypt no banco e retorna 204 No Content
  - AND se as senhas não coincidem, retorna 400 com mensagem "passwords do not match"
  - AND se o usuário já possui senha, retorna 409 Conflict com mensagem "password already set"

- [ ] REQ-2: Alteração de senha
  - GIVEN um usuário autenticado com senha cadastrada
  - WHEN o endpoint `PUT /users/password` é chamado com `current_password`, `new_password` e `new_password_confirmation`
  - THEN a senha atual é verificada, a nova senha é salva como hash bcrypt e retorna 204 No Content
  - AND se a senha atual está incorreta, retorna 401 Unauthorized
  - AND se as novas senhas não coincidem, retorna 400 com mensagem "passwords do not match"

- [ ] REQ-3: Login
  - GIVEN um usuário com senha cadastrada
  - WHEN o endpoint `POST /auth/login` é chamado com `email` e `password`
  - THEN retorna 200 com `access_token` (JWT, curta duração) e `refresh_token` (JWT, longa duração)
  - AND se o email não existe ou senha está incorreta, retorna 401 Unauthorized com mensagem genérica "invalid credentials" (sem revelar se foi email ou senha)
  - AND se o usuário está inativo (active=false), retorna 401 Unauthorized

- [ ] REQ-4: Refresh Token
  - GIVEN um refresh token válido
  - WHEN o endpoint `POST /auth/refresh` é chamado com `refresh_token`
  - THEN retorna 200 com novo `access_token` e novo `refresh_token`
  - AND se o refresh token é inválido/expirado, retorna 401 Unauthorized

- [ ] REQ-5: Middleware JWT
  - GIVEN um endpoint protegido por JWT middleware
  - WHEN uma requisição chega com header `Authorization: Bearer <token>`
  - THEN o token é validado, e `user_id` é extraído e disponibilizado no contexto
  - AND se o token é inválido/expirado/ausente, retorna 401 Unauthorized
  - AND o middleware JWT é aplicado apenas aos endpoints de usuário (CRUD), não aos endpoints de auth nem de role

- [ ] REQ-6: Validação de senha
  - GIVEN qualquer operação que receba senha
  - WHEN a senha tem menos de 8 caracteres, ou menos de 1 letra, ou menos de 1 número, ou menos de 1 caractere especial
  - THEN retorna 400 com mensagem adequada para o erro específico (e.g. "password must be at least 8 characters", "password must contain at least one letter", etc.)

- [ ] REQ-7: Service Key bypass
  - GIVEN o middleware de autenticação
  - WHEN a requisição possui Service Key válida (autenticação service-to-service)
  - THEN o JWT middleware é ignorado (Service Key tem prioridade — mantém compatibilidade com integrações existentes)

- [ ] REQ-8: Login com usuário sem senha
  - GIVEN um usuário existente sem senha cadastrada
  - WHEN o endpoint `POST /auth/login` é chamado com email e qualquer senha
  - THEN retorna 401 Unauthorized com mensagem "invalid credentials" (sem revelar que o usuário existe, mas não tem senha)
  
- [ ] REQ-9: Login com usuário inativo
  - GIVEN um usuário existente com `active=false`
  - WHEN o endpoint `POST /auth/login` é chamado com email e senha corretos
  - THEN retorna 401 Unauthorized com mensagem "invalid credentials" (sem revelar que o usuário é inativo)
  
  - GIVEN a configuração do serviço
  - WHEN `JWTConfig.Enabled` é `true` mas `JWTConfig.Secret` está vazio
  - THEN o serviço deve falhar ao iniciar

## Design

### Architecture Decisions

1. **Senha no domínio User**: A senha é um atributo do User, mas armazenada como hash (nunca em texto plano). Um novo Value Object `Password` encapsula a validação (mínimo 8 chars) e hashing (bcrypt cost 12, mas configurável por variável de ambiente).

2. **JWT como pacote em `pkg/jwt/`**: O JWT é infraestrutura transversal, não pertence ao domínio. Pacote reutilizável com geração e validação de tokens. Usa `golang-jwt/jwt/v5`.

3. **Dois tokens**: Access token (curta duração, 15min default) e Refresh token (longa duração, 7d default). Ambos são JWT — sem armazenamento de sessão em banco/Redis (stateless).

4. **Auth como use cases separados**: `LoginUseCase` e `RefreshUseCase` ficam em `internal/usecases/auth/` — separados dos use cases de User porque pertencem a um contexto funcional diferente (autenticação vs. gestão de usuários).

5. **Password use cases no User domain**: `SetPasswordUseCase` e `ChangePasswordUseCase` ficam em `internal/usecases/user/` porque operam diretamente sobre a entidade User.

6. **Coluna `password_hash`**: Adicionada à tabela `users` como `VARCHAR(255) NULL` (NULL = sem senha cadastrada).

7. **Middleware JWT**: Novo middleware em `internal/infrastructure/web/middleware/jwt.go`. Aplicado seletivamente nos grupos de rotas que exigem autenticação de usuário.

8. **Erros de domínio**: Novos erros específicos para falhas de senha e autenticação, mapeados para códigos HTTP adequados no handler. Erros devem poder ser reaproveitados em outros contextos (e.g. `ErrInvalidCredentials` pode ser usado tanto no login quanto em endpoints protegidos).

### Files to Create

| File                                                                         | Purpose                                              |
|------------------------------------------------------------------------------|------------------------------------------------------|
| `internal/domain/user/vo/password.go`                                        | Value Object Password (validação + hash bcrypt)      |
| `internal/usecases/user/set_password.go`                                     | Use case: cadastrar senha                            |
| `internal/usecases/user/change_password.go`                                  | Use case: alterar senha                              |
| `internal/usecases/user/dto/password.go`                                     | DTOs de entrada/saída para operações de senha        |
| `internal/usecases/auth/login.go`                                            | Use case: login (email + password → JWT)             |
| `internal/usecases/auth/refresh.go`                                          | Use case: refresh token                              |
| `internal/usecases/auth/dto/auth.go`                                         | DTOs de entrada/saída para auth                      |
| `internal/usecases/auth/interfaces/repository.go`                            | Interface do repositório para auth (reusa user repo) |
| `internal/usecases/auth/interfaces/token.go`                                 | Interface do serviço de token JWT                    |
| `pkg/jwt/jwt.go`                                                             | Implementação JWT (geração + validação)              |
| `internal/infrastructure/web/handler/auth.go`                                | Handler HTTP para login e refresh                    |
| `internal/infrastructure/web/handler/password.go`                            | Handler HTTP para set/change password                |
| `internal/infrastructure/web/middleware/jwt.go`                              | Middleware de validação JWT                          |
| `internal/infrastructure/db/postgres/migration/XXXXXX_add_password_hash.sql` | Migration: adicionar coluna password_hash            |
| `config/jwt.go`                                                              | JWTConfig struct e carregamento de env vars          |

### Files to Modify

| File                                                     | Change                                                                                                                                                   |
|----------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| `internal/domain/user/entity.go`                         | Adicionar campo `PasswordHash string`                                                                                                                    |
| `internal/domain/user/errors.go`                         | Adicionar erros: `ErrPasswordAlreadySet`, `ErrInvalidPassword`, `ErrPasswordTooShort`, `ErrPasswordMismatch`, `ErrInvalidCredentials`, `ErrUserInactive` |
| `internal/usecases/user/interfaces/repository.go`        | Adicionar método `UpdatePassword(ctx, id, passwordHash)`                                                                                                 |
| `internal/infrastructure/db/postgres/repository/user.go` | Implementar `UpdatePassword` + incluir `password_hash` no FindByEmail                                                                                    |
| `config/config.go`                                       | Adicionar `JWTConfig` ao `Config` struct                                                                                                                 |
| `cmd/api/server.go`                                      | Wiring dos novos use cases, handlers e middleware JWT                                                                                                    |
| `internal/infrastructure/web/router/router.go`           | Registrar novas rotas (`/auth/login`, `/auth/refresh`, `/users/{id}/password`)                                                                           |

### Dependencies

| Package                        | Purpose                           |
|--------------------------------|-----------------------------------|
| `golang.org/x/crypto/bcrypt`   | Hashing de senha (bcrypt)         |
| `github.com/golang-jwt/jwt/v5` | Geração e validação de tokens JWT |

## Tasks

- [x] TASK-1: Migration — adicionar coluna `password_hash` à tabela `users`
  - Criar migration com `-- +goose Up` adicionando `password_hash VARCHAR(255) NULL` à tabela `users`
  - `-- +goose Down` deve remover a coluna
  - Referência: `internal/infrastructure/db/postgres/migration/20240101002_init_schema.sql`

- [x] TASK-2: Domain — Value Object Password e erros de domínio
  - Criar `internal/domain/user/vo/password.go` com:
    - `NewPassword(plain string) (Password, error)` — valida mínimo 8 chars, faz bcrypt hash (cost 12, mas configurável por env var)
    - `CheckPassword(hash, plain string) error` — verifica senha contra hash
    - Type `Password` (string do hash)
  - Adicionar erros em `internal/domain/user/errors.go`: `ErrPasswordAlreadySet`, `ErrInvalidPassword`, `ErrPasswordTooShort`, `ErrPasswordMismatch`, `ErrInvalidCredentials`, `ErrUserInactive`
  - Adicionar campo `PasswordHash string` à entidade `User` em `entity.go`
  - Verificação: `go build ./internal/domain/...`

- [x] TASK-3: Repository — método `UpdatePassword` e password_hash no mapeamento
  - Adicionar `UpdatePassword(ctx context.Context, id vo.ID, passwordHash string) error` à interface em `internal/usecases/user/interfaces/repository.go`
  - Implementar no repository Postgres: UPDATE users SET password_hash = :hash, updated_at = NOW() WHERE id = :id AND active = true
  - Incluir `password_hash` no mapeamento `userDB` → `User` em `FindByEmail` (e `FindByID` se necessário)
  - Referência: `internal/infrastructure/db/postgres/repository/user.go`
  - Verificação: `go build ./internal/...`

- [x] TASK-4: Use cases — SetPassword e ChangePassword
  - Criar `internal/usecases/user/dto/password.go` com DTOs:
    - `SetPasswordInput{Password, PasswordConfirmation}`
    - `ChangePasswordInput{CurrentPassword, NewPassword, NewPasswordConfirmation}`
  - Criar `internal/usecases/user/set_password.go`:
    - Recebe user ID + SetPasswordInput
    - Busca user, verifica se PasswordHash está vazio (senão ErrPasswordAlreadySet)
    - Valida senhas iguais, cria VO Password, chama repo.UpdatePassword
  - Criar `internal/usecases/user/change_password.go`:
    - Recebe user ID + ChangePasswordInput
    - Busca user, verifica senha atual via VO CheckPassword
    - Valida novas senhas iguais, cria novo hash, chama repo.UpdatePassword
  - Referência: `internal/usecases/user/create.go` para padrão de use case
  - Verificação: `go build ./internal/usecases/...`

- [x] TASK-5: Pacote JWT — `pkg/jwt/`
  - Criar `pkg/jwt/jwt.go` com:
    - `type Claims struct { UserID string; TokenType string ("access"/"refresh") }` embeddando `jwt.RegisteredClaims`
    - `type Service struct { secretKey []byte; accessTTL, refreshTTL time.Duration }`
    - `NewService(secret string, accessTTL, refreshTTL time.Duration) *Service`
    - `GenerateAccessToken(userID string) (string, error)`
    - `GenerateRefreshToken(userID string) (string, error)`
    - `ValidateToken(tokenString string) (*Claims, error)` — valida assinatura, expiração e retorna claims
  - Algoritmo: HS256 (HMAC-SHA256)
  - Verificação: `go build ./pkg/jwt/...`

- [x] TASK-6: Config JWT e dependência
  - Adicionar `JWTConfig` struct em `config/config.go`:
    - `Secret string` (env: `JWT_SECRET`, sem default — obrigatório em produção)
    - `AccessTTL string` (env: `JWT_ACCESS_TTL`, default: `"15m"`)
    - `RefreshTTL string` (env: `JWT_REFRESH_TTL`, default: `"168h"` = 7 dias)
    - `Enabled bool` (env: `JWT_ENABLED`, default: `false`)
  - Adicionar campo `JWT JWTConfig` ao `Config` struct
  - Adicionar validação: se `JWT.Enabled && JWT.Secret == ""` → erro
  - Adicionar `golang.org/x/crypto/bcrypt` e `github.com/golang-jwt/jwt/v5` ao go.mod
  - Verificação: `go build ./config/...`

- [x] TASK-7: Use cases Auth — Login e Refresh
  - Criar `internal/usecases/auth/interfaces/repository.go` — reutiliza o tipo User do domínio:
    - `FindByEmail(ctx, email) (*User, error)`
  - Criar `internal/usecases/auth/interfaces/token.go`:
    - Interface `TokenService` com `GenerateAccessToken`, `GenerateRefreshToken`, `ValidateToken`
  - Criar `internal/usecases/auth/dto/auth.go`:
    - `LoginInput{Email, Password}`, `LoginOutput{AccessToken, RefreshToken}`
    - `RefreshInput{RefreshToken}`, `RefreshOutput{AccessToken, RefreshToken}`
  - Criar `internal/usecases/auth/login.go`:
    - Busca user por email, verifica active, verifica password hash, gera tokens
    - Retorna `ErrInvalidCredentials` genérico para qualquer falha (sem leak de info)
  - Criar `internal/usecases/auth/refresh.go`:
    - Valida refresh token, verifica tipo "refresh", gera novo par de tokens
  - Referência: `internal/usecases/user/create.go` para estrutura
  - Verificação: `go build ./internal/usecases/...`

- [x] TASK-8: Handlers — Auth e Password
  - Criar `internal/infrastructure/web/handler/auth.go`:
    - `POST /auth/login` — bind LoginInput, chama LoginUseCase, retorna tokens
    - `POST /auth/refresh` — bind RefreshInput, chama RefreshUseCase, retorna tokens
  - Criar `internal/infrastructure/web/handler/password.go`:
    - `POST /users/:id/password` — bind SetPasswordInput, chama SetPasswordUseCase
    - `PUT /users/:id/password` — bind ChangePasswordInput, chama ChangePasswordUseCase
  - Atualizar `HandleError` em `error.go` para mapear novos erros de domínio:
    - `ErrPasswordAlreadySet` → 409, `ErrPasswordMismatch/ErrPasswordTooShort` → 400
    - `ErrInvalidCredentials/ErrUserInactive` → 401, `ErrInvalidPassword` → 401
  - Referência: `internal/infrastructure/web/handler/user.go`
  - Verificação: `go build ./internal/infrastructure/...`

- [x] TASK-9: Middleware JWT
  - Criar `internal/infrastructure/web/middleware/jwt.go`:
    - Extrai token do header `Authorization: Bearer <token>`
    - Valida via `pkg/jwt.Service.ValidateToken`
    - Verifica `token_type == "access"`
    - Salva `user_id` no contexto Gin (`c.Set("user_id", claims.UserID)`)
    - Retorna 401 se inválido/ausente/expirado
  - A lógica de bypass por Service Key será tratada no router (aplicação seletiva do middleware)
  - Referência: `internal/infrastructure/web/middleware/service_key.go`
  - Verificação: `go build ./internal/infrastructure/...`

- [x] TASK-10: Router e DI wiring
  - Atualizar `cmd/api/server.go` — `buildDependencies()`:
    - Instanciar `pkg/jwt.Service` com config
    - Instanciar `SetPasswordUseCase`, `ChangePasswordUseCase`
    - Instanciar `LoginUseCase`, `RefreshUseCase`
    - Instanciar auth handler e password handler
    - Criar JWT middleware instance
  - Atualizar router:
    - Grupo `/auth` (público, sem JWT): `POST /login`, `POST /refresh`
    - Grupo `/users` (protegido por JWT quando habilitado): adicionar rotas de password
    - JWT middleware condicional (respeitando `JWT.Enabled` config)
    - Service Key auth continua como camada anterior ao JWT (quem passa por Service Key não precisa de JWT)
  - Verificação: `go build ./cmd/...`

- [x] TASK-11: Testes unitários
  - Testes para `vo/password.go`: hash, validação mínima, check correto/incorreto
  - Testes para `SetPasswordUseCase`: sucesso, senha já cadastrada, senhas não coincidem, senha curta
  - Testes para `ChangePasswordUseCase`: sucesso, senha atual incorreta, senhas não coincidem
  - Testes para `LoginUseCase`: sucesso, email inexistente, senha incorreta, usuário inativo
  - Testes para `RefreshUseCase`: sucesso, token inválido, token tipo errado
  - Testes para `pkg/jwt`: geração, validação, expiração, tipo de token
  - Testes para JWT middleware: token válido, ausente, expirado, tipo errado
  - Seguir padrão de mocks manuais existente (`mocks_test.go` por pacote)
  - Verificação: `make test-unit`

- [x] TASK-12: Swagger annotations
  - Adicionar annotations `@Summary`, `@Tags`, `@Accept`, `@Produce`, `@Param`, `@Success`, `@Failure`, `@Router` para:
    - `POST /auth/login`
    - `POST /auth/refresh`
    - `POST /users/{id}/password`
    - `PUT /users/{id}/password`
  - Adicionar `@SecurityDefinitions.apikey BearerAuth` no main.go
  - Rodar `swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal`
  - Verificação: `make lint`

## Validation Criteria

- [ ] `go build ./...` passes
- [ ] `make lint` passes
- [ ] `make test-unit` passes
- [ ] Migration up/down funciona corretamente
- [ ] Login com credenciais corretas retorna JWT válido
- [ ] Login com credenciais erradas retorna 401 genérico
- [ ] Endpoints protegidos rejetam requisições sem token
- [ ] Endpoints protegidos aceitam requisições com token válido
- [ ] Service Key auth continua funcionando (bypass JWT)
- [ ] Refresh token gera novo par de tokens
- [ ] Senha é armazenada como bcrypt hash (nunca em texto plano)


### Notes

- The spec status is still `DRAFT` — it needs to be set to `APPROVED` before implementation begins.
- `golang.org/x/crypto` exists as indirect dependency in go.mod, but `bcrypt` is not used anywhere yet.
- The user modified the original spec: changed password endpoints from `/users/{id}/password` to `/users/password` (ID from auth context), strengthened password validation (REQ-6 now requires letter + number + special char), added REQ-8 (login without password) and REQ-9 (login inactive + config validation), made bcrypt cost configurable.
- All existing code (user CRUD, service key auth, middleware stack) is unmodified and unaffected.

## Execution Log

### Iteration 1 — TASK-1 (2026-03-30 00:01)

Created migration `internal/infrastructure/db/postgres/migration/20260330001_add_password_hash.sql` adding nullable `password_hash VARCHAR(255)` column to `users` table with reversible down migration.

### Iteration 2 — TASK-2 (2026-03-30 00:02)

Created `vo/password.go` with `NewPassword` (bcrypt hash, configurable cost), `CheckPassword`, and `ValidatePasswordStrength` (8+ chars, letter, number, special). Added password-related errors to `vo/errors.go` and domain errors (`ErrPasswordAlreadySet`, `ErrPasswordMismatch`, `ErrInvalidCredentials`, `ErrUserInactive`) to `errors.go`. Added `PasswordHash` field to User entity.

### Iteration 3 — TASK-3 (2026-03-30 00:03)

Added `password_hash` (sql.NullString) to `userDB` model and updated all SELECT queries (FindByID, FindByEmail, List) to include it. Added `UpdatePassword` method to repository interface and Postgres implementation. Updated `toUser`/`fromDomainUser` mapping.

### Iteration 4 — TASK-4 (2026-03-30 00:04)

Created `dto/password.go` (SetPasswordInput, ChangePasswordInput), `set_password.go` (validates no existing password, checks confirmation match, bcrypt hashes), and `change_password.go` (verifies current password, checks confirmation match, re-hashes). Both use cases support configurable bcrypt cost via builder pattern.

### Iteration 5 — TASK-5 (2026-03-30 00:05)

Created `pkg/jwt/jwt.go` with `Service` (HS256), `GenerateAccessToken`, `GenerateRefreshToken`, `ValidateToken`, custom `Claims` (UserID, TokenType), and error sentinels (`ErrInvalidToken`, `ErrInvalidTokenType`). Added `github.com/golang-jwt/jwt/v5` dependency.

### Iteration 6 — TASK-6 (2026-03-30 00:06)

Added `JWTConfig` struct to `config/config.go` with `Enabled`, `Secret`, `AccessTTL`, `RefreshTTL`, `BcryptCost`. Added env var loading (`JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`, `JWT_BCRYPT_COST`, `JWT_ENABLED`) and startup validation (secret required when enabled, TTL format validation).

### Iteration 7 — TASK-7 (2026-03-30 00:07)

Created `internal/usecases/auth/` with interfaces (`UserRepository`, `TokenService`), DTOs (`LoginInput/Output`, `RefreshInput/Output`), `LoginUseCase` (email lookup, active check, password-less check, bcrypt verify, generic ErrInvalidCredentials), and `RefreshUseCase` (validate token, check type "refresh", generate new pair).

### Iteration 8 — TASK-8 (2026-03-30 00:08)

Created `handler/auth.go` (Login, Refresh with swagger annotations) and `handler/password.go` (SetPassword, ChangePassword — user ID from JWT context). Updated `error.go` translateError with mappings for all new domain errors: password validation → 400, password already set → 409, invalid credentials/inactive → 401.

### Iteration 9 — TASK-9 (2026-03-30 00:09)

Created `middleware/jwt.go` with `JWTAuth` middleware: extracts Bearer token from Authorization header, validates via `pkg/jwt.Service`, checks token type is "access", sets `user_id` in Gin context, returns 401 for any failure.

### Iteration 10 — TASK-10 (2026-03-30 00:10)

Updated `router.go` Dependencies/Config with auth/password handlers and JWT flag. Auth routes (`/auth/login`, `/auth/refresh`) registered as public. User routes protected by JWT middleware when enabled (Service Key group wraps JWT group — REQ-7 bypass). Created `router/auth.go` and `router/password.go`. Updated `server.go` buildDependencies to wire JWT service, password use cases, and auth use cases with conditional JWT enablement.

### Iteration 11 — TASK-11 (2026-03-30 00:11)

Created unit tests across all new packages: `vo/password_test.go` (12 cases: hash, validation, check), `set_password_test.go` (5 cases), `change_password_test.go` (3 cases), `login_test.go` (6 cases: success, not found, wrong password, inactive, no password, invalid email), `refresh_test.go` (3 cases), `pkg/jwt/jwt_test.go` (6 cases: generate, validate, expired, wrong signature, malformed), `middleware/jwt_test.go` (6 cases). Added `UpdatePassword` to MockRepository. All 5 packages pass.

### Iteration 12 — TASK-12 (2026-03-30 00:12)

Added `@securityDefinitions.apikey BearerAuth` to `cmd/api/doc.go`. Swagger annotations already in place from TASK-8 handlers. Regenerated swagger docs via `swag init` — all new endpoints (auth/login, auth/refresh, users/password POST/PUT) appear in generated docs.

<!-- Ralph Loop appends here automatically — do not edit manually -->
