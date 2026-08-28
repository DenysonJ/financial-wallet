# SPEC-account-001: Limite de Crédito da Account

- **Status:** implemented
- **Tipo:** feature
- **Módulo:** account
> **Entidades de domínio:** [[account]] (ver `specs/domain/`)
- **Arquitetura:** clean
- **Stack:** go
- **Atualizado em:** 2026-08-21

## 1. Objetivo de Negócio

Cartão de crédito só faz sentido com um teto: o usuário precisa saber quanto do cartão já
comprometeu e quanto ainda pode gastar. Hoje a account de tipo `credit_card` guarda apenas o
saldo devedor, sem nenhuma referência de limite — o número isolado não responde "isso é
muito ou pouco?".

Esta spec adiciona o limite de crédito à account: um valor cadastral, informado pelo usuário
na criação do cartão e editável depois. É o dado que faltava para calcular percentual usado
e remanescente.

## 2. Contexto e Escopo

- **Ator:** usuário autenticado (JWT), sobre as próprias accounts.
- **Gatilho:** endpoints REST de account já existentes — `POST /accounts`,
  `PUT /accounts/:id`, `GET /accounts`, `GET /accounts/:id`.
- **Natureza:** atributo cadastral. O limite **não** participa de nenhuma movimentação
  financeira: não altera saldo, não gera lançamento, não é debitado nem creditado.
- **Motivação imediata:** esta spec é o desbloqueio de
  [SPEC-dashboard-001 §3.4](dashboard-summary.spec.md) (bloco `creditUsage`), que a declarou
  como dependência externa bloqueante.
- **Premissa de dados:** não há accounts em produção. Isso permite tornar o limite
  obrigatório para cartões sem backfill (ver RN-22).

### Fora de escopo

- **Bloqueio de lançamentos por limite.** O limite é informativo: `POST` de statement,
  importação OFX e estorno continuam aceitando débito que ultrapasse o limite. Nenhum
  arquivo do fluxo de statement é alterado por esta spec.
- Histórico/auditoria das alterações de limite (quem mudou, quando, de quanto para quanto).
- Limite de cheque especial para `bank_account` — o campo é exclusivo de `credit_card`.
- Fatura: data de fechamento, vencimento, juros, encargos, parcelamento, pagamento mínimo.
- Filtro, ordenação ou busca por `credit_limit` nos endpoints de listagem.
- Alteração do `type` de uma account existente (o update atual já não permite).
- Alertas ou notificação ao se aproximar do limite.
- Multi-moeda — o valor é centavos da moeda única implícita, como `balance`.

## 3. Regras de Negócio (EARS)

### 3.1 Invariantes da entidade Account

- **INV-01:** Account de tipo `credit_card` DEVE ter limite de crédito definido.
- **INV-02:** Account de tipo diferente de `credit_card` DEVE ter limite de crédito ausente.
- **INV-03:** O limite, quando presente, DEVE ser um inteiro de centavos **maior que zero** e
  menor ou igual a `1.000.000.000.000` (R$ 10 bilhões) — teto que impede valor absurdo e
  garante que a soma dos limites de todas as accounts do usuário não estoure `int64`.
- **INV-04:** O limite DEVE ser independente do saldo: nenhuma alteração de limite muda
  `balance`, e nenhum lançamento muda o limite.
- **INV-05:** NÃO DEVE ser possível construir uma Account que viole INV-01, INV-02 ou
  INV-03 — a construção falha em vez de produzir entidade inválida.

### 3.2 Criação de account

- **RN-01:** QUANDO uma account de tipo `credit_card` for criada sem limite, o sistema DEVE
  recusar com `CREDIT_LIMIT_REQUIRED`.
- **RN-02:** QUANDO uma account de tipo diferente de `credit_card` for criada com limite
  informado — inclusive zero —, o sistema DEVE recusar com `CREDIT_LIMIT_NOT_ALLOWED`.
- **RN-03:** SE o limite informado for menor ou igual a zero, ou maior que o teto de
  INV-03, ENTÃO o sistema DEVE recusar com `INVALID_CREDIT_LIMIT`.
- **RN-04:** SE o limite não for um inteiro (fracionário, texto, booleano), ENTÃO o sistema
  DEVE recusar com `INVALID_CREDIT_LIMIT`, sem arredondar nem truncar.
- **RN-05:** QUANDO a criação for aceita, o sistema DEVE persistir o limite e manter o saldo
  inicial em zero — limite não é saldo (INV-04).

### 3.3 Atualização de account

- **RN-06:** O sistema DEVE aceitar `credit_limit` como campo isolado na atualização
  parcial, sem exigir `name` nem `description` no mesmo corpo.
- **RN-07:** QUANDO um limite válido for informado numa account `credit_card`, o sistema DEVE
  substituir o valor anterior.
- **RN-08:** SE for informado limite numa account de tipo diferente de `credit_card`, ENTÃO o
  sistema DEVE recusar com `CREDIT_LIMIT_NOT_ALLOWED`.
- **RN-09:** O sistema DEVE tratar `credit_limit` **ausente** e `credit_limit: null` como
  "não alterar". Remover o limite de um `credit_card` é proibido por INV-01, e nenhum outro
  tipo pode ter limite para remover — não existe operação de limpeza.
- **RN-10:** O sistema NÃO DEVE permitir alterar o `type` da account, preservando INV-01 e
  INV-02 sem necessidade de revalidação cruzada.
- **RN-11:** O sistema DEVE permitir reduzir o limite para um valor **menor que o saldo
  devedor atual** — limite é dado cadastral, e travar a edição impediria o usuário de
  corrigir um valor digitado errado antes de pagar a fatura.
- **RN-12:** SE a account não existir ou não pertencer ao usuário autenticado, ENTÃO o
  sistema DEVE responder `ACCOUNT_NOT_FOUND`, sem distinguir os dois casos.
- **RN-13:** QUANDO o limite for alterado, o sistema DEVE atualizar `updated_at` e NÃO DEVE
  gerar statement nem alterar `balance`.

### 3.4 Leitura e contrato

- **RN-14:** O sistema DEVE expor `credit_limit` nas respostas de `GET /accounts`,
  `GET /accounts/:id` e `PUT /accounts/:id`, com valor `null` para os tipos sem limite.
- **RN-15:** O sistema DEVE serializar `credit_limit` como **número inteiro de centavos** no
  JSON REST, igual ao campo `balance` já existente. (Divergência intencional do scalar
  `Money` string da SPEC-dashboard-001: cada protocolo mantém sua convenção.)
- **RN-16:** O sistema NÃO DEVE criar permissão nova: `account:write` para criar,
  `account:update` para alterar e `account:read` para ler, como hoje.

### 3.5 Interface com a consolidação

- **RN-17:** ONDE a consolidação de uso de crédito estiver disponível, o sistema DEVE
  representar uso acima do limite sem saturar: percentual usado acima de 100% e valor
  disponível negativo.
- **RN-18:** A consolidação DEVE ler o limite vigente na account (snapshot), não um
  histórico — não existe limite "na data".

### 3.6 Migração de esquema

- **RN-19:** A migration DEVE adicionar a coluna com tipo explícito `BIGINT NULL` e uma
  constraint `CHECK` que garanta INV-01 e INV-02 no banco, como rede de segurança —
  a validação primária continua sendo a da entidade.
- **RN-20:** A migration DEVE ter seções `Up` e `Down` simétricas: o `Down` remove a
  constraint e a coluna, sem perda de outros dados.
- **RN-21:** SE existir, no momento da migration, alguma account `credit_card` sem limite,
  ENTÃO a migration DEVE **falhar** — nunca preencher com valor arbitrário, porque inventar
  limite de crédito é inventar dado financeiro.
- **RN-22:** O sistema NÃO DEVE incluir backfill de dados na migration, conforme a premissa
  de ausência de dados em produção declarada na seção 2.

### Tabela de decisão — tipo × limite informado

| `type`         | Limite informado    | Resultado                    | Regra            |
|----------------|---------------------|------------------------------|------------------|
| `credit_card`  | válido (> 0, ≤ teto)| Aceita                       | RN-01, RN-07     |
| `credit_card`  | ausente na criação  | `CREDIT_LIMIT_REQUIRED` 422  | RN-01, INV-01    |
| `credit_card`  | ausente no update   | Mantém o valor atual         | RN-09            |
| `credit_card`  | ≤ 0 ou > teto       | `INVALID_CREDIT_LIMIT` 400   | RN-03, INV-03    |
| `bank_account` | qualquer valor      | `CREDIT_LIMIT_NOT_ALLOWED` 422 | RN-02, RN-08, INV-02 |
| `bank_account` | ausente             | Aceita, limite `null`        | INV-02           |
| `cash`         | qualquer valor      | `CREDIT_LIMIT_NOT_ALLOWED` 422 | RN-02, RN-08, INV-02 |
| `cash`         | ausente             | Aceita, limite `null`        | INV-02           |

## 4. Casos de Uso

### UC-01: Cadastrar cartão com limite — exercita RN-01, RN-05, INV-01

- **Dado** um usuário autenticado com permissão `account:write`
- **Quando** ele cria uma account `type: "credit_card"` com `credit_limit: 500000` (R$ 5.000)
- **Então** a account é criada com saldo zero e limite de 500000, e o `GET` subsequente
  devolve `credit_limit: 500000`

### UC-02: Cartão sem limite é recusado — exercita RN-01, INV-01

- **Dado** um usuário autenticado
- **Quando** ele cria uma account `type: "credit_card"` sem o campo `credit_limit`
- **Então** recebe 422 `CREDIT_LIMIT_REQUIRED` e nenhuma account é criada

### UC-03: Limite em conta corrente é recusado — exercita RN-02, INV-02

- **Dado** um usuário autenticado
- **Quando** ele cria uma account `type: "bank_account"` com `credit_limit: 100000`
- **Então** recebe 422 `CREDIT_LIMIT_NOT_ALLOWED` e nenhuma account é criada

### UC-04: Limite inválido — exercita RN-03, RN-04, INV-03

- **Dado** um usuário autenticado
- **Quando** ele cria um `credit_card` com `credit_limit` igual a `0`, a `-1`, a `1500.75` ou
  acima do teto de R$ 10 bilhões
- **Então** recebe 400 `INVALID_CREDIT_LIMIT` em todos os casos, sem arredondamento

### UC-05: Ajustar o limite do cartão — exercita RN-06, RN-07, RN-13

- **Dado** um `credit_card` com limite de R$ 5.000
- **Quando** o usuário envia `PUT /accounts/:id` com apenas `{"credit_limit": 800000}`
- **Então** o limite passa a R$ 8.000, `updated_at` é atualizado, `balance` permanece
  inalterado e nenhum statement é gerado

### UC-06: Limite em conta de dinheiro existente — exercita RN-08, INV-02

- **Dado** uma account `type: "cash"` já cadastrada
- **Quando** o usuário tenta atualizar informando `credit_limit: 50000`
- **Então** recebe 422 `CREDIT_LIMIT_NOT_ALLOWED` e nada é alterado

### UC-07: Reduzir o limite abaixo da fatura — exercita RN-11, RN-17

- **Dado** um `credit_card` com limite de R$ 5.000 e saldo de −R$ 3.000 (fatura aberta)
- **Quando** o usuário reduz o limite para R$ 1.000
- **Então** a alteração é aceita, e a consolidação de crédito passa a mostrar 300% usado com
  valor disponível de −R$ 2.000, sem saturar em 100%

### UC-08: Campo omitido não apaga o limite — exercita RN-09

- **Dado** um `credit_card` com limite de R$ 5.000
- **Quando** o usuário envia `PUT /accounts/:id` com apenas `{"name": "Cartão Nubank"}`, ou
  com `{"credit_limit": null}`
- **Então** o nome muda (no primeiro caso) e o limite permanece R$ 5.000 nos dois

### UC-09: Cartão de outro usuário — exercita RN-12

- **Dado** o id de um `credit_card` pertencente a outro usuário
- **Quando** o usuário tenta alterar o limite desse cartão
- **Então** recebe 404 `ACCOUNT_NOT_FOUND`, sem informação que revele a existência da conta

### UC-10: Leitura de contas mistas — exercita RN-14, RN-15

- **Dado** um usuário com um `credit_card` (limite R$ 5.000), uma `bank_account` e uma `cash`
- **Quando** ele consulta `GET /accounts`
- **Então** o cartão traz `"credit_limit": 500000` como número inteiro, e as outras duas
  contas traem `"credit_limit": null`

## 5. API / Contrato

Nenhuma rota nova. As rotas abaixo ganham um campo.

| Item        | Valor |
|-------------|-------|
| Método/Rota | `POST /accounts` · `PUT /accounts/:id` · `GET /accounts` · `GET /accounts/:id` |
| Auth        | `Authorization: Bearer <jwt>` + `Service-Name`/`Service-Key` (quando habilitado) |
| Permissões  | `account:write` (criar) · `account:update` (alterar) · `account:read` (ler) |

**Entrada — `POST /accounts` (campo novo):**

```
credit_limit  int64  obrigatório se type=credit_card, proibido nos outros tipos
                     centavos, > 0 e <= 1000000000000
```

```json
{
  "name": "Cartão Nubank",
  "type": "credit_card",
  "description": "cartão principal",
  "credit_limit": 500000
}
```

**Entrada — `PUT /accounts/:id` (campo novo, atualização parcial):**

```
credit_limit  int64  opcional; ausente ou null = não alterar (RN-09)
                     proibido quando type != credit_card
```

**Saída (sucesso) — `GET /accounts/:id`:**

```json
{
  "data": {
    "id": "019...",
    "user_id": "019...",
    "name": "Cartão Nubank",
    "type": "credit_card",
    "description": "cartão principal",
    "balance": -300000,
    "credit_limit": 500000,
    "active": true,
    "created_at": "2026-08-21T10:00:00Z",
    "updated_at": "2026-08-21T10:00:00Z"
  }
}
```

`POST /accounts` mantém a resposta atual (`id` + `created_at`), sem o campo novo.
`GET /accounts` repete o objeto acima dentro de `data[]`.

## 6. Condições de Erro

Formato padrão do serviço: `{"errors": {"message": "..."}}`.

| Código                     | HTTP | Quando ocorre                                                        | Regra                    |
| -------------------------- | ---- | -------------------------------------------------------------------- | ------------------------ |
| `CREDIT_LIMIT_REQUIRED`    | 422  | `credit_card` criado sem limite                                       | RN-01, INV-01            |
| `CREDIT_LIMIT_NOT_ALLOWED` | 422  | Limite informado em `bank_account` ou `cash` (criação ou atualização) | RN-02, RN-08, INV-02     |
| `INVALID_CREDIT_LIMIT`     | 400  | Limite ≤ 0, acima do teto, ou não inteiro                             | RN-03, RN-04, INV-03     |
| `ACCOUNT_NOT_FOUND`        | 404  | Account inexistente ou de outro usuário                               | RN-12                    |
| `FORBIDDEN`                | 403  | Falta `account:write` ou `account:update`                             | RN-16                    |
| `UNAUTHORIZED`             | 401  | JWT ausente, expirado ou inválido                                     | RN-16                    |
| `INTERNAL_ERROR`           | 500  | Falha de persistência; mensagem sanitizada, sem SQL nem stack          | RN-19                    |

**Critério dos status:** 400 para valor malformado em si (alinhado a
`ErrInvalidAmount`/`ErrInvalidID`, já mapeados como 400); 422 para regra de negócio que
depende do **estado** da entidade — a combinação tipo × limite (alinhado a
`ErrAccountNotActive`, já 422).

## 7. Notas de Arquitetura

### 7.1 Onde cada regra mora (Clean Architecture)

- **Value Object `vo.CreditLimit`** (`internal/domain/account/vo/`) — valida INV-03 no
  construtor e implementa `driver.Valuer`/`sql.Scanner`, seguindo o padrão já estabelecido
  por `vo.AccountType` e `statement/vo.Amount`. Zero dependência externa. É o guardião do
  **valor**.
- **Entidade `Account`** — guarda a invariante **cruzada** (tipo × limite, INV-01/INV-02).
  Ela não cabe no VO, que não conhece o tipo da conta, nem no use case, que é regra da
  aplicação: "cartão tem teto, dinheiro em espécie não" é regra corporativa e pertence ao
  núcleo. O limite é representado como ponteiro (`*vo.CreditLimit`), porque ausência é um
  estado legítimo e o zero-value do VO é inválido por INV-03.
- **Construção segura (INV-05)** — `NewAccount` passa a receber o limite e a retornar
  `(*Account, error)`, deixando de poder produzir entidade inválida; a alteração ganha um
  método próprio na entidade, que recusa tipo incompatível. Isso muda a assinatura atual, que
  não retorna erro; o único chamador de produção é
  [create.go:53](internal/usecases/account/create.go:53).
- **Use cases (`create`, `update`)** — apenas orquestram: convertem o input em VO, chamam a
  entidade e propagam o erro de domínio. **Não repetem** a validação — regra duplicada em
  duas camadas é regra que vai divergir.
- **Erros de domínio puros** — `ErrCreditLimitRequired` e `ErrCreditLimitNotAllowed` no
  pacote de domínio, `ErrInvalidCreditLimit` no pacote `vo`. Nenhum carrega status HTTP.
- **Tradução HTTP só no handler** — os três sentinelas entram em `domainErrors`
  (`handler/error.go`), a fonte única de verdade que também popula
  `apperror.DomainSentinels`. Registrar ali é o que faz o span classificar esses erros como
  **esperados** (`WarnSpan`) em vez de alertar como falha.
- **Telemetria** — `WarnSpan` para as recusas de validação e ownership, `OkSpan` no caminho
  feliz, sempre **no use case**; handler e repositório não tocam span.

### 7.2 Persistência (guia Go, seção 5)

- `accountDB` ganha `CreditLimit *int64` mapeado em `db:"credit_limit"`; `NULL` no banco
  vira ponteiro nulo no domínio, sem sentinela mágica como `-1` ou `0`.
- `INSERT`, `UPDATE` e `SELECT` parametrizados, sem concatenação.
- Migration Goose: `ALTER TABLE accounts ADD COLUMN credit_limit BIGINT NULL` mais
  `CHECK` nomeada expressando INV-01/INV-02 (`type = 'credit_card'` ⇔ limite não nulo),
  com `Down` simétrico. Tipo explícito, sem depender de default do Postgres.
- **Nenhum índice novo**: não há filtro, ordenação ou join por `credit_limit` (a
  consolidação lê o campo junto da linha da account). Criar índice aqui seria custo de
  escrita sem leitura que o justifique.
- A `CHECK` é defesa em profundidade, não a validação primária: regra de negócio no banco
  não é testável em unidade nem produz erro semântico de domínio.

### 7.3 Contrato e o tri-state do JSON

`*int64` não distingue "campo ausente" de `null` explícito — ambos decodificam como ponteiro
nulo. Em vez de introduzir um tipo customizado com `json.RawMessage` só para representar a
remoção, RN-09 **elimina a necessidade**: como INV-01 proíbe cartão sem limite e INV-02
proíbe limite nos outros tipos, não existe operação de limpeza a expressar. A regra escolhida
é a que mantém o contrato simples sem perder capacidade.

### 7.4 Testes (guia Go, seção 3)

- **VO:** table-driven cobrindo zero, negativo, teto exato, teto + 1 e valor típico.
- **Entidade:** matriz completa da tabela de decisão de 3.6 (3 tipos × limite presente,
  ausente, inválido) — é o teste que prova INV-01, INV-02 e INV-05 sem banco nem mock.
- **Use cases:** mocks de repositório gerados por **mockery**, reaproveitando o mock de
  account já existente em `internal/mocks/`; cobrir cada linha da seção 6.
- **Repositório:** `go-sqlmock` verificando que o `NULL` faz round-trip para ponteiro nulo.
- **E2E:** TestContainers cobrindo os dois 422 condicionais (que dependem do tipo persistido)
  e a redução de limite abaixo da fatura (UC-07).
- Regenerar Swagger (`swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal`)
  — a CI falha se `docs/` divergir.

## 8. Glossário (termos novos)

- **Limite de crédito (`creditLimit`):** teto de gasto de uma account `credit_card`, em
  centavos. Dado cadastral informado pelo usuário; não é saldo e não é movimentado.
- **Saldo devedor / valor usado:** parte do limite já comprometida, derivada do saldo
  negativo do cartão (`max(0, -balance)`). Não é campo armazenado.
- **Uso acima do limite (overlimit):** estado em que o valor usado excede o limite vigente —
  possível quando o limite é reduzido depois dos gastos (RN-11). Representado com percentual
  acima de 100% e disponível negativo, nunca saturado.
- **Teto de limite:** valor máximo aceito para o campo (R$ 10 bilhões), guarda contra input
  absurdo e overflow na soma agregada (INV-03).

## 9. Rastreabilidade

- **Implementação:**
  - `internal/domain/account/vo/credit_limit.go` (VO `CreditLimit` + `MaxCreditLimit`)
  - `internal/domain/account/vo/errors.go` (`ErrInvalidCreditLimit`)
  - `internal/domain/account/errors.go` (`ErrCreditLimitRequired`, `ErrCreditLimitNotAllowed`)
  - `internal/domain/account/entity.go` (campo `CreditLimit`, `AcceptsCreditLimit`,
    `NewAccount` agora retorna `(*Account, error)`, `SetCreditLimit`, `resolveCreditLimit`)
  - `internal/usecases/account/{create,update,get,list}.go` + `mapper.go`
  - `internal/usecases/account/dto/{create,update,get}.go`
  - `internal/infrastructure/db/postgres/repository/account.go`
  - `internal/infrastructure/web/handler/{account,error}.go`
  - `internal/infrastructure/db/postgres/migration/20260823232448_add_credit_limit_to_accounts.sql`
- **Testes:**
  - `internal/domain/account/vo/credit_limit_test.go` (faixa do valor, `Value`/`Scan`)
  - `internal/domain/account/credit_limit_test.go` (matriz tipo × limite, `SetCreditLimit`)
  - `internal/usecases/account/{create_test.go,update_test.go}`
  - `internal/infrastructure/db/postgres/repository/account_test.go` (round-trip `NULL`)
  - `internal/infrastructure/web/handler/account_credit_limit_test.go` (status HTTP)
  - `tests/e2e/account_credit_limit_test.go` (ciclo completo + `CHECK` do banco)
- **Doc:** `docs/modules/account-credit-limit.md`;
  `docs/swagger.{json,yaml}` + `docs/docs.go` regenerados por `swag init`;
  `docs/financial-wallet.postman_collection.json` com os requests
  "Create Credit Card Account" e "Update Credit Card Limit"
- **Validação:** `go build ./...` ✓ · `gofmt` ✓ · `go vet ./...` ✓ ·
  `make test-unit` ✓ (todos verdes) · `tests/e2e/account_credit_limit_test.go` ✓ contra
  Postgres real · migration testada `Up → Down → Up` no Postgres local ✓

### Cobertura de regras

| Regra | Status | Onde é verificada |
| ----- | ------ | ----------------- |
| INV-01 | ✓ | `credit_limit_test.go` (matriz) · `account_credit_limit_test.go` (e2e, `CHECK`) |
| INV-02 | ✓ | `credit_limit_test.go` (matriz) · e2e `CHECK` |
| INV-03 | ✓ | `vo/credit_limit_test.go` (0, negativo, teto, teto+1) · e2e `CHECK` |
| INV-04 | ✓ | `SetCreditLimit` (saldo intacto) · `..._CreditLimitDoesNotTouchBalance` · e2e |
| INV-05 | ✓ | matriz assere `a == nil` em todo caso de erro |
| RN-01 | ✓ | domínio + use case + handler (422) + e2e |
| RN-02 | ✓ | domínio + use case + handler (422) + e2e |
| RN-03 | ✓ | VO + use case + handler (400) + e2e |
| RN-04 | ✓ (ver nota) | handler: fracionário e textual → 400 |
| RN-05 | ✓ | `TestNewAccount_CreditCardStartsWithZeroBalance` + e2e (coluna persistida) |
| RN-06 | ✓ | use case + handler + e2e (corpo só com `credit_limit`) |
| RN-07 | ✓ | `SetCreditLimit` + use case + e2e |
| RN-08 | ✓ | domínio + use case + handler (422) + e2e |
| RN-09 | ✓ | use case (omitido) + handler (`null`) + e2e (ambos) |
| RN-10 | ✓ | `TestAccountHandler_Update_IgnoresType` |
| RN-11 | ✓ | domínio + use case + handler + e2e (limite < saldo devedor) |
| RN-12 | ✓ | e2e (cartão de outro usuário → 404) + testes de ownership existentes |
| RN-13 | ✓ | `SetCreditLimit` (`updated_at`) + `..._DoesNotTouchBalance` + e2e (saldo intacto) |
| RN-14 | ✓ | handler `GetByID` (valor e `null`) + e2e |
| RN-15 | ✓ | handler assere `"credit_limit":500000` como inteiro |
| RN-16 | ✓ por construção | nenhuma permissão nova: rotas e seeds de RBAC inalterados |
| RN-17 | ⟳ delegada | estado de *overlimit* alcançável e persistido (e2e UC-07); a representação percentual é da SPEC-dashboard-001 |
| RN-18 | ⟳ delegada | leitura do snapshot é da SPEC-dashboard-001; aqui não há histórico a manter |
| RN-19 | ✓ | migration + `TestE2E_AccountCreditLimit_DatabaseConstraint` (3 violações recusadas) |
| RN-20 | ✓ | `Up → Down → Up` executado no Postgres local |
| RN-21 | ✓ por construção | a `CHECK` valida as linhas existentes ao ser criada, então a migration falha em vez de preencher valor |
| RN-22 | ✓ por construção | migration não tem `UPDATE`/`INSERT` de backfill |

**Nota sobre RN-04:** valor não inteiro é recusado com **400**, como a regra exige, mas a
mensagem é a genérica de bind do projeto (`invalid request body`), não
`INVALID_CREDIT_LIMIT` — o JSON nem chega a virar `*int64`. Optou-se por manter a mensagem
padrão em vez de inspecionar `json.UnmarshalTypeError` no handler, que é o comportamento já
aplicado aos outros campos numéricos do serviço (ex.: `amount` em statement). O requisito
essencial da regra — não arredondar nem truncar — está garantido e testado.

### Impacto em SPEC-dashboard-001

Esta spec **desbloqueia** o bloco `creditUsage` (§3.4 da spec do dashboard). Como o limite
passou a ser **obrigatório** para `credit_card`, três regras de lá precisam ser ajustadas na
próxima revisão daquela spec:

| Regra afetada | Situação | Ajuste necessário |
|---|---|---|
| **RN-33** (`limitDefined = false`, percentuais nulos) | Inalcançável para dado novo — todo cartão tem limite | Manter apenas como defesa contra linha inconsistente; deixar de ser caminho de negócio esperado |
| **RN-34** (`accountsWithoutLimit`) | Sempre `0` | Manter no contrato por compatibilidade, documentando que só é diferente de zero em dado legado/inconsistente |
| **UC-09** (cartão "sem limite cadastrado") | Cenário não reproduzível | Reescrever usando o cenário de **overlimit** do UC-07 desta spec |

O cálculo de **SPEC-dashboard-001 RN-31** (`usedAmount = max(0, -balance)`) foi
**confirmado** contra o código: o ledger soma em `credit` e subtrai em `debit`, então cartão
com fatura aberta tem saldo negativo. **SPEC-dashboard-001 RN-32** já produz percentual acima
de 100% e disponível negativo sem cap, portanto está compatível com RN-17 desta spec —
nenhuma alteração necessária lá.
