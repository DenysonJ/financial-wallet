package e2e

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/DenysonJ/financial-wallet/pkg/cache/redisclient"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
)

var testDB *sqlx.DB
var testCache *redisclient.RedisClient

// PostgresContainer encapsula o container do Postgres para testes
type PostgresContainer struct {
	*postgres.PostgresContainer
	ConnectionString string
}

// CreatePostgresContainer cria e inicia um container Postgres para testes
func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	return &PostgresContainer{
		PostgresContainer: container,
		ConnectionString:  connStr,
	}, nil
}

// getMigrationsDir retorna o caminho absoluto para o diretório de migrations
func getPostgresMigrationsDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(currentFile)
	// Navega de tests/e2e/ para a raiz do projeto e depois para o diretório de migrations
	return filepath.Join(testDir, "..", "..", "internal", "infrastructure", "db", "postgres", "migration")
}

// RunMigrations executa as migrações no banco de teste usando goose
func RunPostgresMigrations(db *sql.DB) error {
	// Configurar goose
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Executar todas as migrations
	if err := goose.Up(db, getPostgresMigrationsDir()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// CreateRedisContainer cria e inicia um container Redis para testes
func CreateRedisContainer(ctx context.Context) (testcontainers.Container, string, error) {
	container, err := redis.Run(ctx,
		"redis:7-alpine",
		redis.WithSnapshotting(10, 1),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start redis container: %w", err)
	}

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get redis connection string: %w", err)
	}

	return container, connStr, nil
}

// GetTestCache retorna o cache Redis de teste
func GetTestCache() *redisclient.RedisClient {
	return testCache
}

// TestMain configura o ambiente de teste e2e
func TestMain(m *testing.M) {
	ctx := context.Background()

	// Iniciar container Postgres
	pgContainer, err := CreatePostgresContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to create postgres container: %v", err)
	}

	// Conectar ao banco
	testDB, err = sqlx.Connect("postgres", pgContainer.ConnectionString)
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Executar migrações usando goose
	if migrateErr := RunPostgresMigrations(testDB.DB); migrateErr != nil {
		log.Fatalf("Failed to run migrations: %v", migrateErr)
	}

	// Remove sobras de uma execução anterior interrompida, restrito ao namespace
	// reservado (@e2e.local) — nunca toca dado fora dele.
	if purgeErr := purgeE2ENamespace(); purgeErr != nil {
		log.Fatalf("Failed to purge e2e namespace: %v", purgeErr)
	}

	// Iniciar container Redis
	redisContainer, redisConnStr, err := CreateRedisContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to create redis container: %v", err)
	}

	// Criar cliente Redis para testes
	testCache, err = redisclient.NewRedisClient(redisclient.RedisConfig{
		URL:     redisConnStr,
		TTL:     "5m",
		Enabled: true,
	})
	if err != nil {
		log.Fatalf("Failed to create redis client: %v", err)
	}

	// Definir variáveis de ambiente para a aplicação
	os.Setenv("DB_DSN", pgContainer.ConnectionString)
	os.Setenv("REDIS_URL", redisConnStr)
	os.Setenv("REDIS_ENABLED", "true")

	// Executar testes
	code := m.Run()

	// Cleanup
	if testCache != nil {
		testCache.Close()
	}
	testDB.Close()
	if err := redisContainer.Terminate(ctx); err != nil {
		log.Printf("Failed to terminate redis container: %v", err)
	}
	if err := pgContainer.Terminate(ctx); err != nil {
		log.Printf("Failed to terminate postgres container: %v", err)
	}

	os.Exit(code)
}

// HTTPClient retorna um http.Client configurado para testes
func HTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

// GetTestDB retorna a conexão do banco de teste
func GetTestDB() *sqlx.DB {
	return testDB
}

// =============================================================================
// Escopo de dados dos testes
//
// Nenhum teste apaga tabela inteira. Cada teste trabalha num namespace próprio e
// remove apenas o que criou:
//
//   - o dono dos dados é um user novo por teste (UUID v7), e todo o resto do
//     schema pendura nele — accounts → statements → statement_tags, além de
//     categories, tags e user_roles, que têm user_id;
//   - nomes e e-mails carregam o token do escopo e o domínio reservado
//     e2e.local, então as listagens podem ser filtradas pelo próprio escopo em
//     vez de depender de tabela vazia;
//   - a limpeza roda em t.Cleanup, portanto acontece mesmo quando o teste falha.
//
// O resultado: a suíte não depende de ordem de execução, um teste nunca apaga o
// dado de outro e nada fora do namespace e2e é tocado — inclusive se a suíte for
// apontada para um banco compartilhado em vez do container efêmero.
// =============================================================================

// e2eDomain é o domínio de e-mail reservado aos dados de teste. Só a suíte cria
// registros com ele, e a limpeza de segurança do TestMain só remove esses.
const e2eDomain = "e2e.local"

// newScopeToken devolve um token curto e único para identificar um teste. Usa o
// segmento aleatório do UUID v7, o que evita colisão entre testes criados no
// mesmo milissegundo.
func newScopeToken() string {
	return vo.NewID().String()[24:]
}

// scopedEmail monta um e-mail dentro do namespace reservado.
func scopedEmail(token, local string) string {
	return fmt.Sprintf("%s-%s@%s", local, token, e2eDomain)
}

// scopedName prefixa um nome com o token do escopo, para que a listagem possa
// ser filtrada por ?name=<token> e a contagem seja exata sem limpar a tabela.
func scopedName(token, label string) string {
	return fmt.Sprintf("%s %s", token, label)
}

// cleanupUserData remove tudo que pertence aos users informados, na ordem das
// FKs. Chame via t.Cleanup para que rode também quando o teste falha.
func cleanupUserData(t *testing.T, userIDs ...string) {
	t.Helper()
	if len(userIDs) == 0 {
		return
	}

	ids := pq.Array(userIDs)
	statements := []string{
		`DELETE FROM statement_tags WHERE statement_id IN (
			SELECT s.id FROM statements s
			JOIN accounts a ON a.id = s.account_id
			WHERE a.user_id = ANY($1))`,
		`DELETE FROM statements WHERE account_id IN (SELECT id FROM accounts WHERE user_id = ANY($1))`,
		`DELETE FROM accounts WHERE user_id = ANY($1)`,
		`DELETE FROM categories WHERE user_id = ANY($1)`,
		`DELETE FROM tags WHERE user_id = ANY($1)`,
		`DELETE FROM user_roles WHERE user_id = ANY($1)`,
		`DELETE FROM users WHERE id = ANY($1)`,
	}

	for _, query := range statements {
		if _, execErr := testDB.Exec(query, ids); execErr != nil {
			t.Errorf("cleanup do escopo falhou: %v\nquery: %s", execErr, query)
		}
	}
}

// cleanupScope remove os dados de um escopo quando os users foram criados pela
// própria API (o teste não conhece os IDs de antemão): resolve os users pelo
// token embutido no e-mail e delega para cleanupUserData.
func cleanupScope(t *testing.T, token string) {
	t.Helper()

	var userIDs []string
	selectErr := testDB.Select(&userIDs,
		`SELECT id FROM users WHERE email LIKE $1`, "%-"+token+"@"+e2eDomain)
	if selectErr != nil {
		t.Errorf("cleanup do escopo %s falhou ao resolver users: %v", token, selectErr)
		return
	}

	cleanupUserData(t, userIDs...)
}

// cleanupRolesByPrefix remove apenas os roles criados por um escopo. Role é
// entidade global (não tem user_id), e as migrations semeiam os roles `admin` e
// `user` de que o RBAC depende — apagar a tabela inteira destruiria esse seed.
func cleanupRolesByPrefix(t *testing.T, token string) {
	t.Helper()
	if _, execErr := testDB.Exec(`DELETE FROM roles WHERE name LIKE $1`, token+"%"); execErr != nil {
		t.Errorf("cleanup de roles falhou: %v", execErr)
	}
}

// purgeE2ENamespace roda uma vez no início da suíte e remove sobras de execuções
// anteriores interrompidas (Ctrl+C, timeout, crash). Restringe-se ao namespace
// reservado: users com e-mail @e2e.local e a subárvore deles. É inofensivo no
// container efêmero e é a rede de segurança caso a suíte aponte para um banco
// com dados de verdade.
func purgeE2ENamespace() error {
	const selectLeftovers = `SELECT id FROM users WHERE email LIKE $1`

	var leftovers []string
	if selectErr := testDB.Select(&leftovers, selectLeftovers, "%@"+e2eDomain); selectErr != nil {
		return fmt.Errorf("listing e2e leftovers: %w", selectErr)
	}
	if len(leftovers) == 0 {
		return nil
	}

	log.Printf("e2e: removendo %d user(s) de execução anterior no namespace @%s", len(leftovers), e2eDomain)

	ids := pq.Array(leftovers)
	for _, query := range []string{
		`DELETE FROM statement_tags WHERE statement_id IN (
			SELECT s.id FROM statements s
			JOIN accounts a ON a.id = s.account_id
			WHERE a.user_id = ANY($1))`,
		`DELETE FROM statements WHERE account_id IN (SELECT id FROM accounts WHERE user_id = ANY($1))`,
		`DELETE FROM accounts WHERE user_id = ANY($1)`,
		`DELETE FROM categories WHERE user_id = ANY($1)`,
		`DELETE FROM tags WHERE user_id = ANY($1)`,
		`DELETE FROM user_roles WHERE user_id = ANY($1)`,
		`DELETE FROM users WHERE id = ANY($1)`,
	} {
		if _, execErr := testDB.Exec(query, ids); execErr != nil {
			return fmt.Errorf("purging e2e namespace: %w", execErr)
		}
	}
	return nil
}
