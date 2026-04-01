package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server      ServerConfig
	DB          DBConfig
	Otel        OtelConfig
	Redis       RedisConfig
	Auth        AuthConfig
	JWT         JWTConfig
	Swagger     SwaggerConfig
	Idempotency IdempotencyConfig
	RateLimit   RateLimitConfig
}

type RateLimitConfig struct {
	// Enabled indica se o rate limiting está habilitado. Requer Redis.
	Enabled bool
	// Requests é o número máximo de requisições por janela (rotas gerais).
	Requests int
	// Window é a duração da janela para rotas gerais. Ex: "1m"
	Window string
	// AuthRequests é o número máximo de requisições por janela para rotas de auth.
	AuthRequests int
	// AuthWindow é a duração da janela para rotas de auth. Ex: "1m"
	AuthWindow string
}

type JWTConfig struct {
	// Enabled indica se a autenticação JWT está habilitada.
	// Quando true, Secret é obrigatório.
	Enabled bool
	// Secret é a chave HMAC para assinar e verificar tokens JWT.
	Secret string
	// AccessTTL é a duração do access token. Ex: "15m"
	AccessTTL string
	// RefreshTTL é a duração do refresh token. Ex: "168h" (7 dias)
	RefreshTTL string
	// BcryptCost é o custo do bcrypt para hashing de senhas. Default: 12
	BcryptCost int
}

type AuthConfig struct {
	// Enabled indica se a autenticação está habilitada.
	// Em HML/PRD deve ser true. Se true e ServiceKeys vazio → fail-closed (503).
	Enabled bool
	// ServiceKeys no formato "service1:key1,service2:key2"
	ServiceKeys string
}

type ServerConfig struct {
	Port        string
	Env         string
	GinMode     string // "release", "debug", "test" — default: "" (debug)
	MaxBodySize int64  // Max request body size in bytes (0 = default 1MB)
}

type IdempotencyConfig struct {
	Enabled bool
	TTL     string // Duração do armazenamento de respostas. Ex: "24h"
	LockTTL string // Duração do lock de processamento. Ex: "30s"
}

// DBConfig contém a configuração do banco de dados.
// Writer (primary) usa DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE.
// Replica (read) usa DB_REPLICA_* com fallback para os valores do writer.
type DBConfig struct {
	// Writer (primary)
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string

	// Writer pool
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// Replica (read)
	ReplicaEnabled         bool
	ReplicaHost            string
	ReplicaPort            string
	ReplicaUser            string
	ReplicaPassword        string
	ReplicaName            string
	ReplicaSSLMode         string
	ReplicaMaxOpenConns    int
	ReplicaMaxIdleConns    int
	ReplicaConnMaxLifetime time.Duration
	ReplicaConnMaxIdleTime time.Duration
}

// GetWriterDSN builds the writer (primary) connection DSN in key=value format.
func (c *DBConfig) GetWriterDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

// GetReaderDSN builds the reader (replica) connection DSN.
// Empty fields fall back to writer values.
func (c *DBConfig) GetReaderDSN() string {
	host := c.ReplicaHost
	if host == "" {
		host = c.Host
	}
	port := c.ReplicaPort
	if port == "" {
		port = c.Port
	}
	user := c.ReplicaUser
	if user == "" {
		user = c.User
	}
	password := c.ReplicaPassword
	if password == "" {
		password = c.Password
	}
	name := c.ReplicaName
	if name == "" {
		name = c.Name
	}
	sslMode := c.ReplicaSSLMode
	if sslMode == "" {
		sslMode = c.SSLMode
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, name, sslMode)
}

type OtelConfig struct {
	ServiceName  string
	CollectorURL string
	Insecure     bool
}

type RedisConfig struct {
	URL          string
	TTL          string // ex: "5m", "1h"
	Enabled      bool
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type SwaggerConfig struct {
	Enabled bool
	Host    string
}

// Load configura a aplicação lendo do ambiente.
// Prioridade:
// 1. Variáveis de Ambiente (maior prioridade)
// 2. Arquivo .env (desenvolvimento local)
// 3. Defaults (fallback seguro)
func Load() (*Config, error) {
	// Carrega .env se existir (ignora erro se não existir)
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			Env:         getEnv("APP_ENV", "development"),
			GinMode:     getEnv("GIN_MODE", ""),
			MaxBodySize: int64(getEnvInt("HTTP_MAX_BODY_SIZE", 1<<20)), // default 1MB
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "user"),
			Password: getEnv("DB_PASSWORD", "password"),
			Name:     getEnv("DB_NAME", "users"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),

			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 90*time.Second),

			ReplicaEnabled:         getEnvBool("DB_REPLICA_ENABLED", false),
			ReplicaHost:            os.Getenv("DB_REPLICA_HOST"),
			ReplicaPort:            os.Getenv("DB_REPLICA_PORT"),
			ReplicaUser:            os.Getenv("DB_REPLICA_USER"),
			ReplicaPassword:        os.Getenv("DB_REPLICA_PASSWORD"),
			ReplicaName:            os.Getenv("DB_REPLICA_NAME"),
			ReplicaSSLMode:         os.Getenv("DB_REPLICA_SSLMODE"),
			ReplicaMaxOpenConns:    getEnvInt("DB_REPLICA_MAX_OPEN_CONNS", 40),
			ReplicaMaxIdleConns:    getEnvInt("DB_REPLICA_MAX_IDLE_CONNS", 20),
			ReplicaConnMaxLifetime: getEnvDuration("DB_REPLICA_CONN_MAX_LIFETIME", 5*time.Minute),
			ReplicaConnMaxIdleTime: getEnvDuration("DB_REPLICA_CONN_MAX_IDLE_TIME", 90*time.Second),
		},
		Otel: OtelConfig{
			ServiceName:  getEnv("OTEL_SERVICE_NAME", "financial-wallet"),
			CollectorURL: getEnv("OTEL_COLLECTOR_URL", ""),
			Insecure:     getEnvBool("OTEL_INSECURE", true),
		},
		Redis: RedisConfig{
			URL:          getEnv("REDIS_URL", "redis://localhost:6379"),
			TTL:          getEnv("REDIS_TTL", "5m"),
			Enabled:      getEnvBool("REDIS_ENABLED", false),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 30),
			MinIdleConns: getEnvInt("REDIS_MIN_IDLE_CONNS", 5),
			DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 500*time.Millisecond),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 200*time.Millisecond),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 200*time.Millisecond),
		},
		Auth: AuthConfig{
			Enabled:     getEnvBool("SERVICE_KEYS_ENABLED", false),
			ServiceKeys: getEnv("SERVICE_KEYS", ""),
		},
		JWT: JWTConfig{
			Enabled:    getEnvBool("JWT_ENABLED", false),
			Secret:     os.Getenv("JWT_SECRET"),
			AccessTTL:  getEnv("JWT_ACCESS_TTL", "15m"),
			RefreshTTL: getEnv("JWT_REFRESH_TTL", "168h"),
			BcryptCost: getEnvInt("JWT_BCRYPT_COST", 12),
		},
		Swagger: SwaggerConfig{
			Enabled: getEnvBool("SWAGGER_ENABLED", false),
			Host:    getEnv("SWAGGER_HOST", ""),
		},
		Idempotency: IdempotencyConfig{
			Enabled: getEnvBool("IDEMPOTENCY_ENABLED", false),
			TTL:     getEnv("IDEMPOTENCY_TTL", "24h"),
			LockTTL: getEnv("IDEMPOTENCY_LOCK_TTL", "30s"),
		},
		RateLimit: RateLimitConfig{
			Enabled:      getEnvBool("RATE_LIMIT_ENABLED", false),
			Requests:     getEnvInt("RATE_LIMIT_REQUESTS", 100),
			Window:       getEnv("RATE_LIMIT_WINDOW", "1m"),
			AuthRequests: getEnvInt("RATE_LIMIT_AUTH_REQUESTS", 10),
			AuthWindow:   getEnv("RATE_LIMIT_AUTH_WINDOW", "1m"),
		},
	}, nil
}

// Validate checks for invalid configuration states at startup.
// Returns an error if a critical misconfiguration is detected.
func (c *Config) Validate() error {
	// DB: sslmode=disable in non-dev environments
	if c.Server.Env != "development" && c.DB.SSLMode == "disable" {
		fmt.Println("WARNING: DB_SSLMODE=disable in non-development environment")
	}

	// Idempotency: enabled but Redis disabled
	if c.Idempotency.Enabled && !c.Redis.Enabled {
		return fmt.Errorf("IDEMPOTENCY_ENABLED=true requires REDIS_ENABLED=true")
	}

	// Idempotency: validate TTL strings are parseable
	if c.Idempotency.Enabled {
		if _, parseErr := time.ParseDuration(c.Idempotency.TTL); parseErr != nil {
			return fmt.Errorf("IDEMPOTENCY_TTL=%q is not a valid duration: %w", c.Idempotency.TTL, parseErr)
		}
		if _, parseErr := time.ParseDuration(c.Idempotency.LockTTL); parseErr != nil {
			return fmt.Errorf("IDEMPOTENCY_LOCK_TTL=%q is not a valid duration: %w", c.Idempotency.LockTTL, parseErr)
		}
	}

	// JWT: enabled but no secret
	if c.JWT.Enabled && c.JWT.Secret == "" {
		return fmt.Errorf("JWT_ENABLED=true requires JWT_SECRET to be set")
	}

	// JWT: validate TTL strings are parseable
	if c.JWT.Enabled {
		if _, parseErr := time.ParseDuration(c.JWT.AccessTTL); parseErr != nil {
			return fmt.Errorf("JWT_ACCESS_TTL=%q is not a valid duration: %w", c.JWT.AccessTTL, parseErr)
		}
		if _, parseErr := time.ParseDuration(c.JWT.RefreshTTL); parseErr != nil {
			return fmt.Errorf("JWT_REFRESH_TTL=%q is not a valid duration: %w", c.JWT.RefreshTTL, parseErr)
		}
		// BcryptCost: must be greater than 8 and less than 64
		if c.JWT.BcryptCost < 8 || c.JWT.BcryptCost > 64 {
			return fmt.Errorf("JWT_BCRYPT_COST should be between 8 and 64, got %d", c.JWT.BcryptCost)
		}
	}

	// RateLimit: enabled but Redis disabled
	if c.RateLimit.Enabled && !c.Redis.Enabled {
		return fmt.Errorf("RATE_LIMIT_ENABLED=true requires REDIS_ENABLED=true")
	}

	// RateLimit: validate window durations are parseable
	if c.RateLimit.Enabled {
		if _, parseErr := time.ParseDuration(c.RateLimit.Window); parseErr != nil {
			return fmt.Errorf("RATE_LIMIT_WINDOW=%q is not a valid duration: %w", c.RateLimit.Window, parseErr)
		}
		if _, parseErr := time.ParseDuration(c.RateLimit.AuthWindow); parseErr != nil {
			return fmt.Errorf("RATE_LIMIT_AUTH_WINDOW=%q is not a valid duration: %w", c.RateLimit.AuthWindow, parseErr)
		}
	}

	// MaxBodySize: must be positive if set
	if c.Server.MaxBodySize < 0 {
		return fmt.Errorf("HTTP_MAX_BODY_SIZE must be >= 0, got %d", c.Server.MaxBodySize)
	}

	return nil
}

// getEnv retorna o valor da variável de ambiente ou o fallback se não existir.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvBool retorna o valor booleano da variável de ambiente ou o fallback.
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		parsed, parseErr := strconv.ParseBool(value)
		if parseErr != nil {
			return fallback
		}
		return parsed
	}
	return fallback
}

// getEnvInt retorna o valor inteiro da variável de ambiente ou o fallback.
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return fallback
		}
		return parsed
	}
	return fallback
}

// getEnvDuration retorna o valor de duração da variável de ambiente ou o fallback.
// Aceita formatos como "5m", "1h", "30s".
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		parsed, parseErr := time.ParseDuration(value)
		if parseErr != nil {
			return fallback
		}
		return parsed
	}
	return fallback
}
