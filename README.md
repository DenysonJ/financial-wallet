# Financial Wallet Microservice

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![Architecture](https://img.shields.io/badge/Architecture-Clean-blueviolet)](docs/adr/001-clean-architecture.md)
[![CI](https://img.shields.io/badge/CI-GitHub_Actions-2088FF?logo=github-actions)](https://github.com/DenysonJ/financial-wallet/actions)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker)](docker/)

**Micro-serviço de finanças pessoais com Clean Architecture, PostgreSQL, Redis e OpenTelemetry.**

---

## Quick Start

### 1. Clone

```bash
git clone https://github.com/DenysonJ/financial-wallet
cd financial-wallet
```

### 2. Configure

```bash
cp .env.example .env
# Editar .env com suas configs
make setup
```

### 3. Desenvolva

```bash
make dev          # Hot reload local (Go + DB + Redis)
make test         # Testes
make lint         # Linters
make run          # Tudo em Docker (sem Go local)
```

---

## Comandos

```bash
make help              # Lista todos os comandos com descrições

# Desenvolvimento
make setup             # Setup completo (tools + Docker + migrations)
make dev               # Servidor local com hot reload
make run               # Tudo em Docker (infra + migrations + API)
make run-stop          # Para todos os containers
make changelog         # Gera sugestão de changelog a partir dos commits

# Qualidade
make lint              # golangci-lint (v2) + gofmt + goimports
make vulncheck         # Varredura de vulnerabilidades (govulncheck)
make swagger           # Regenera documentação Swagger
make mocks             # Regenera mocks com mockery

# Testes
make test              # Todos (unit + E2E)
make test-unit         # Apenas unit tests
make test-e2e          # E2E com TestContainers
make test-coverage     # Relatório HTML de cobertura

# Infraestrutura
make docker-up         # Sobe PostgreSQL + Redis
make docker-down       # Para containers
make observability-up  # ELK + OTel Collector
make observability-setup # Dashboard + alertas no Kibana

# Load Tests
make load-smoke        # Validação básica (5 VUs)
make load-test         # Carga progressiva (até 50 VUs)
make load-stress       # Encontrar limites (até 200 VUs)
```

---

## Configuração

Hierarquia (maior prioridade primeiro):

1. **Variáveis de Ambiente** — Kubernetes, CI/CD
2. **Arquivo `.env`** — Desenvolvimento local
3. **Defaults no código** — Fallback seguro

```bash
# Servidor
SERVER_PORT=8080

# Postgres (Writer)
DB_HOST=localhost
DB_PORT=5432
DB_USER=user
DB_PASSWORD=password
DB_NAME=users

# Redis
REDIS_ENABLED=true
REDIS_URL=redis://localhost:6379

# JWT
JWT_ENABLED=true
JWT_SECRET=your-secret-key
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

# Swagger (desabilitado por padrão — habilite para desenvolvimento)
SWAGGER_ENABLED=true

# Service Key Auth (vazio = modo desenvolvimento)
SERVICE_KEYS=myservice:sk_myservice_abc123
```

Ver `.env.example` para a lista completa e [ADR-003](docs/adr/003-config-strategy.md) para detalhes.


### Autenticação

A API suporta duas formas de autenticação:

#### JWT (principal)

Autentique via `/auth/login` e use o token nas requisições:

```bash
# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "Str0ng!Pass1"}'

# Usar o access_token retornado
curl -X GET http://localhost:8080/users \
  -H "Authorization: Bearer <access_token>"

# Refresh quando o token expirar
curl -X POST http://localhost:8080/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

#### Service Key (service-to-service)

Para comunicação entre serviços, use headers `X-Service-Name` e `X-Service-Key`:

```bash
curl -X GET http://localhost:8080/users \
  -H "X-Service-Name: financial-wallet" \
  -H "X-Service-Key: sk_financial_wallet_abc123"
```

#### Rotas e permissões

| Rota                           | Proteção                   | Permissão        |
|--------------------------------|----------------------------|------------------|
| `/health`, `/ready`            | Pública                    | —                |
| `/swagger/*`                   | Pública                    | —                |
| `/auth/login`, `/auth/refresh` | Pública (rate limited)     | —                |
| `POST /users`                  | Service Key ou JWT         | `user:write`     |
| `GET /users`, `GET /users/:id` | Service Key ou JWT         | `user:read`      |
| `PUT /users/:id`               | Service Key ou JWT         | `user:write`     |
| `DELETE /users/:id`            | Service Key ou JWT         | `user:delete`    |
| `POST /accounts`               | JWT                        | `account:write`  |
| `GET /accounts`                | JWT                        | `account:read`   |
| `GET /accounts/:id`            | JWT                        | `account:read`   |
| `PUT /accounts/:id`            | JWT                        | `account:update` |
| `DELETE /accounts/:id`         | JWT                        | `account:delete` |
| `/roles/*`                     | Service Key ou JWT (admin) | `role:*`         |

**RBAC**: roles `admin` (todas as permissões) e `user` (read/write nas próprias contas e dados).

**Comportamento por ambiente:**

| Ambiente        | `SERVICE_KEYS_ENABLED`  | `JWT_ENABLED`           | Resultado                                 |
|-----------------|-------------------------|-------------------------|-------------------------------------------|
| Desenvolvimento | `false` (padrão)        | `true`                  | JWT ativo, service key bypass             |
| HML/PRD         | `true`                  | `true`                  | Ambos ativos, validação completa          |
| HML/PRD         | `true`                  | `true` + keys **vazio** | **503 Service Unavailable** (fail-closed) |

### CI/CD

| Feature            | O que faz                                                           | Onde roda                       |
|--------------------|---------------------------------------------------------------------|---------------------------------|
| **GitHub Actions** | Lint (golangci-lint v2) + testes + coverage (Codecov) + govulncheck | PRs para `main`/`develop`       |
| **Dependabot**     | PRs automáticos para atualizar dependências Go e Actions            | Semanal (Go) / Mensal (Actions) |

### Qualidade automatizada

| Feature                    | O que faz                                                                               | Quando roda     |
|----------------------------|-----------------------------------------------------------------------------------------|-----------------|
| **Testes unitários + E2E** | Unit + sqlmock + mockery + E2E com TestContainers                                       | `make test`     |
| **golangci-lint v2**       | 15+ linters incluindo gosec, gocritic, errorlint                                        | Pre-commit + CI |
| **govulncheck**            | Varredura de vulnerabilidades em dependências                                           | Pre-push + CI   |
| **Mockery**                | Geração automática de mocks para todas as interfaces                                    | `make mocks`    |
| **Lefthook**               | 3 camadas: pre-commit (formatação), commit-msg (convenção), pre-push (lint+testes+vuln) | Automático      |
| **Codecov**                | Upload de cobertura com relatório em PRs                                                | CI              |

### DevOps

| Feature             | O que faz                                      | Comando                 |
|---------------------|------------------------------------------------|-------------------------|
| **Docker Compose**  | DB + Redis + API tudo em Docker                | `make run`              |
| **Hot Reload**      | Air com rebuild automático                     | `make dev`              |
| **Observabilidade** | ELK 8.13 + OTel + dashboards + alertas         | `make observability-up` |
| **Load Tests**      | k6 com 4 cenários (smoke, load, stress, spike) | `make load-smoke`       |
| **Migrations**      | Goose SQL bidirecional                         | `make migrate-up`       |

---

## Estrutura do projeto

O código é organizado em **camadas com responsabilidades claras**. O domínio fica no centro, protegido de detalhes de infraestrutura — exatamente o padrão de dependência da Clean Architecture.

```text
               ┌─────────────────────────────┐
               │      Infrastructure         │
               │  (Banco, Cache, HTTP, OTel) │
               │                             │
               │   ┌─────────────────────┐   │
               │   │     Use Cases       │   │
               │   │ (Operações de       │   │
               │   │  negócio, 1 por     │   │
               │   │  arquivo)           │   │
               │   │                     │   │
               │   │   ┌─────────────┐   │   │
               │   │   │   Domain    │   │   │
               │   │   │ (Entidades, │   │   │
               │   │   │  VOs, Erros)│   │   │
               │   │   └─────────────┘   │   │
               │   └─────────────────────┘   │
               └─────────────────────────────┘

Dependências apontam para dentro: Infrastructure → Use Cases → Domain
Domain não conhece nada das camadas externas.
```

### Na prática, no código

```text
├── cmd/
│   ├── api/              # Entrypoint HTTP server
│   └── migrate/          # Binário de migrations (K8s Job)
├── config/               # Configuração (godotenv + env vars)
├── internal/
│   ├── domain/           # Entidades, Value Objects, erros (zero deps externas)
│   │   ├── user/         # User aggregate (entity, VOs: Email, Password)
│   │   ├── account/      # Account aggregate (entity, VO: AccountType)
│   │   └── role/         # Role aggregate
│   ├── usecases/         # Casos de uso + interfaces (1 arquivo por operação)
│   ├── mocks/            # Mocks gerados pelo mockery
│   └── infrastructure/   # Banco, cache, HTTP handlers, telemetria
├── pkg/                  # Pacotes reutilizáveis entre serviços
│   ├── apperror/         # Erros estruturados
│   ├── cache/            # Redis + singleflight
│   ├── database/         # DB Writer/Reader (driver-agnostic)
│   ├── httputil/         # Respostas padronizadas + wrappers Gin (httpgin/)
│   ├── idempotency/      # Idempotência distribuída
│   ├── logutil/          # Logging + mascaramento de dados pessoais
│   ├── telemetry/        # OpenTelemetry setup
│   └── vo/               # Value Objects compartilhados (ID UUID v7)
├── docker/               # Dockerfile + docker-compose + observabilidade
├── docs/                 # ADRs + guias
└── tests/                # E2E (TestContainers) + load (k6)
```

### Arquitetura de infraestrutura

```text
                    ┌─────────────────┐
                    │    Ingress      │
                    │   (NGINX)       │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │   API Service   │
                    │   (Go + Gin)    │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼────────┐ ┌───▼───┐ ┌───────▼───────┐
     │   PostgreSQL    │ │ Redis │ │ OTel Collector│
     │   (Dados)       │ │(Cache)│ │ (Telemetria)  │
     └─────────────────┘ └───────┘ └───────────────┘
```

### Pacotes reutilizáveis (pkg/)

Estes pacotes podem ser importados por **qualquer serviço Go** — não só quem usa o template:

| Pacote            | O que faz                                                                                        |
|-------------------|--------------------------------------------------------------------------------------------------|
| `pkg/vo`          | Value Object ID compartilhado (UUID v7) — usado por todos os domínios                            |
| `pkg/apperror`    | Erros estruturados com código, mensagem e status HTTP                                            |
| `pkg/httputil`    | Respostas JSON padronizadas (`WriteSuccess`, `WriteError`) + wrappers Gin em `httputil/httpgin/` |
| `pkg/cache`       | Interface de cache + Redis + singleflight (proteção contra stampede)                             |
| `pkg/database`    | Conexão de banco driver-agnostic (`database/sql`) com Writer/Reader cluster                      |
| `pkg/idempotency` | Idempotência distribuída via Redis (lock/unlock, fingerprint SHA-256)                            |
| `pkg/logutil`     | Logging estruturado com propagação de contexto e mascaramento de PII                             |
| `pkg/telemetry`   | Setup OTel (traces + HTTP metrics + DB pool metrics)                                             |
| `pkg/health`      | Health checker com verificação de dependências e timeouts                                        |

---

## Ferramentas de DX

#### Skills (slash commands)

| Skill                   | O que faz                                                              |
|-------------------------|------------------------------------------------------------------------|
| `/spec`                 | Gera especificação estruturada (SDD) com requisitos, design e tasks    |
| `/ralph-loop`           | Execução autônoma task-by-task a partir de uma spec                    |
| `/spec-review`          | Valida implementação contra os requisitos da spec                      |
| `/new-endpoint`         | Scaffold de endpoint seguindo Clean Architecture                       |
| `/fix-issue`            | Fluxo completo de bug fix (entender → planejar → implementar → testar) |
| `/validate`             | Pipeline de validação (build, lint, testes)                            |
| `/review`               | Code review (arquitetura + segurança + convenções)                     |
| `/full-review-team`     | Review paralelo com 3 agentes (arquitetura + segurança + DB)           |
| `/security-review-team` | Auditoria de segurança paralela com 3 especialistas                    |
| `/debug-team`           | Investigação paralela de bugs com hipóteses concorrentes               |
| `/migrate`              | Gerenciamento de migrations Goose                                      |
| `/load-test`            | Testes de carga com k6                                                 |

#### SDD + Ralph Loop — Desenvolvimento Orientado a Especificação

Para features complexas, o projeto oferece um fluxo spec-driven com execução autônoma:

```text
/spec "Add audit logging to user write operations"
  → Gera .specs/user-audit-log.md (requisitos, design, tasks)
  → Você revisa e aprova

/ralph-loop .specs/user-audit-log.md
  → Executa task por task autonomamente
  → Stop hook controla iteração (exit code 2)
  → Validação completa roda no final

/spec-review .specs/user-audit-log.md
  → Verifica implementação contra requisitos
```

A spec é agnóstica de arquitetura — funciona tanto com camadas separadas quanto colapsadas. Ver [guia completo](docs/guides/sdd-ralph-loop.md).

#### Hooks (qualidade automática)

| Hook | Quando roda | O que faz |
| ---- | ----------- | --------- |
| `guard-bash.sh` | Antes de comandos bash | Bloqueia `.env` staging, `git add -A`, DROP, `--no-verify` |
| `lint-go-file.sh` | Após editar arquivo Go | goimports + gopls diagnostics |
| `validate-migration.sh` | Após editar migration | Garante seções Up + Down |
| `ralph-loop.sh` | Ao finalizar tarefa | Controla iteração do Ralph Loop |
| `stop-validate.sh` | Ao finalizar tarefa | Gate de qualidade: build + lint + testes |

#### Agentes Especializados

3 agentes com memória persistente, usados pelos skills de review e debug:

- **code-reviewer** — Compliance de arquitetura, idiomas Go, padrões do projeto
- **security-reviewer** — OWASP Top 10, injeção, auth, dados sensíveis
- **db-analyst** — Schema, performance de queries, migrations, pool

Para mais detalhes sobre a configuração de IA, ver [CLAUDE.md](CLAUDE.md).

---

### Via VS Code

Abra o projeto no VS Code com a extensão **Dev Containers** instalada. Ele detecta o `.devcontainer/devcontainer.json` automaticamente e oferece "Reopen in Container".

### Via Makefile (sem VS Code)

```bash
make sandbox          # Abre um shell no container com firewall ativo
make sandbox-claude   # Abre o Claude Code direto no container
make sandbox-shell    # Conecta num container já rodando
make sandbox-stop     # Para o container
make sandbox-firewall # Testa se o firewall está funcionando
make sandbox-status   # Mostra status do container e volumes
```

### O que vem instalado no container

- Go 1.25 + todas as dev tools (air, goose, lefthook, golangci-lint, swag, gopls, goimports, mockery)
- Node.js 20 + Claude Code
- Docker-in-Docker (para rodar `docker compose` dentro do container)
- zsh com Powerline10k
- git-delta para diffs aprimorados

### Firewall (default-deny)

O container roda com `--cap-add=NET_ADMIN` e um script de firewall (`init-firewall.sh`) que:

1. Bloqueia **todo** tráfego de saída por padrão
2. Permite apenas domínios necessários: Anthropic (Claude), GitHub, Go modules, Docker Hub, Kibana
3. Permite tráfego local (host network, Docker network)

Isso garante que o Claude Code com `--dangerously-skip-permissions` não consiga acessar serviços externos não autorizados.

---

## Documentação

O projeto inclui 8 ADRs (Architecture Decision Records) em `docs/adr/` explicando o **porquê** de cada decisão técnica, e guias práticos em `docs/guides/`:

| Guia                                               | Sobre                                                      |
|----------------------------------------------------|------------------------------------------------------------|
| [architecture.md](docs/guides/architecture.md)     | Diagramas e visão geral                                    |
| [cache.md](docs/guides/cache.md)                   | Cache com Redis, singleflight e pool config                |
| [sdd-ralph-loop.md](docs/guides/sdd-ralph-loop.md) | SDD + Ralph Loop — fluxo spec-driven com execução autônoma |

Para agentes de IA, ver [AGENTS.md](AGENTS.md) e [CLAUDE.md](CLAUDE.md).

---

## Roadmap

O app está em constante evolução. Próximas features planejadas:

- [ ] **Statements (Registros financeiros)** — CRUD de lançamentos (receitas/despesas) vinculados a accounts, com categorização e data de competência
- [ ] **Parser de arquivos OFX** — Import automático de extratos bancários no formato OFX (Open Financial Exchange) para popular statements
- [ ] **Dashboard de resumo** — Endpoints para consolidação: saldo por account, totais por categoria, fluxo mensal
- [ ] **Orçamentos (Orçamentos)** — Definição de limites mensais por categoria com alerta de ultrapassagem
- [ ] **Tags e categorias** — Sistema flexível de categorização de statements com tags customizáveis
- [ ] **Export CSV/PDF** — Geração de relatórios exportáveis para uso externo

---

## FAQ

