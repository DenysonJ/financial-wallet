package router

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/handler"
	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/DenysonJ/financial-wallet/internal/usecases/auth/interfaces"
	"github.com/DenysonJ/financial-wallet/pkg/health"
	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/idempotency"
	"github.com/DenysonJ/financial-wallet/pkg/ratelimit"
	"github.com/DenysonJ/financial-wallet/pkg/telemetry"
)

// Config contém configurações do router
type Config struct {
	ServiceName        string
	ServiceKeysEnabled bool   // fail-closed em HML/PRD se keys vazio
	ServiceKeys        string // "service1:key1,service2:key2"
	SwaggerEnabled     bool
	RateLimitEnabled   bool
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	RateLimitAuthReqs  int
	RateLimitAuthWin   time.Duration
	// TrustedProxies is a comma-separated list of CIDR ranges trusted to set
	// X-Forwarded-For (e.g. "10.0.0.0/8,192.168.0.0/16"). Empty = trust none
	TrustedProxies string
	Env            string
}

// Dependencies agrupa todas as dependências necessárias para o router
type Dependencies struct {
	HealthChecker    *health.Checker
	UserHandler      *handler.UserHandler
	RoleHandler      *handler.RoleHandler
	AccountHandler   *handler.AccountHandler
	StatementHandler *handler.StatementHandler
	AuthHandler      *handler.AuthHandler
	PasswordHandler  *handler.PasswordHandler
	JWTService       interfaces.TokenService
	PermissionLoader middleware.PermissionLoader
	HTTPMetrics      *telemetry.HTTPMetrics
	IdempotencyStore idempotency.Store
	RateLimitStore   ratelimit.Store
	Config           Config
}

// Setup configura e retorna o router Gin com todos os middlewares e rotas
func Setup(deps Dependencies) *gin.Engine {
	r := gin.New()

	// Gin defaults to trusting every CIDR; restrict so X-Forwarded-For from
	// untrusted hops can't spoof c.ClientIP for rate-limit/idempotency keys.
	configureTrustedProxies(r, deps.Config.TrustedProxies)

	// Recovery middleware (panic recovery)
	r.Use(gin.Recovery())

	// OpenTelemetry (must be before Logger to populate trace_id)
	r.Use(otelgin.Middleware(deps.Config.ServiceName))

	// HTTP Metrics (count, duration, Apdex)
	r.Use(middleware.Metrics(deps.HTTPMetrics))

	// Custom structured logger
	r.Use(middleware.Logger())

	// Idempotency (optional — only if store is provided)
	if deps.IdempotencyStore != nil {
		r.Use(middleware.Idempotency(deps.IdempotencyStore))
	}

	// Rate Limit — global (applied before auth, after logging/metrics)
	if deps.Config.RateLimitEnabled && deps.RateLimitStore != nil {
		r.Use(middleware.RateLimit(middleware.RateLimitConfig{
			Store:  deps.RateLimitStore,
			Limit:  deps.Config.RateLimitRequests,
			Window: deps.Config.RateLimitWindow,
		}))
	}

	// Public routes (no auth required)
	if deps.Config.SwaggerEnabled {
		registerSwaggerRoutes(r)
	}
	registerHealthRoutes(r, deps)

	// Auth routes (public — no service key or JWT required, but stricter rate limit)
	if deps.AuthHandler != nil {
		authGroup := r.Group("/auth")
		if deps.Config.RateLimitEnabled && deps.RateLimitStore != nil {
			authGroup.Use(middleware.RateLimit(middleware.RateLimitConfig{
				Store:      deps.RateLimitStore,
				Limit:      deps.Config.RateLimitAuthReqs,
				Window:     deps.Config.RateLimitAuthWin,
				FailClosed: true,
			}))
		}
		RegisterAuthRoutes(authGroup, deps.AuthHandler)
	}

	// Protected routes (auth required if SERVICE_KEYS is configured)
	authConfig := middleware.ServiceKeyConfig{
		Enabled: deps.Config.ServiceKeysEnabled,
		Keys:    middleware.ParseServiceKeys(deps.Config.ServiceKeys),
	}
	protected := r.Group("")
	protected.Use(middleware.ServiceKeyAuth(authConfig))

	// Set Password: Service Key only
	if deps.PasswordHandler != nil {
		RegisterSetPasswordRoute(protected, deps.PasswordHandler, deps.PermissionLoader)
	}

	// User routes + Change Password: JWT authentication
	jwtProtected := protected.Group("")
	jwtProtected.Use(middleware.JWTAuth(deps.JWTService))
	RegisterUserRoutes(jwtProtected, deps.UserHandler, deps.PermissionLoader)
	if deps.PasswordHandler != nil {
		RegisterChangePasswordRoute(jwtProtected, deps.PasswordHandler, deps.PermissionLoader)
	}

	// Account routes: JWT authentication
	if deps.AccountHandler != nil {
		accountGroup := protected.Group("")
		accountGroup.Use(middleware.JWTAuth(deps.JWTService))
		RegisterAccountRoutes(accountGroup, deps.AccountHandler, deps.PermissionLoader)
	}

	// Statement routes: JWT authentication (nested under accounts)
	if deps.StatementHandler != nil {
		statementGroup := protected.Group("")
		statementGroup.Use(middleware.JWTAuth(deps.JWTService))
		RegisterStatementRoutes(statementGroup, deps.StatementHandler, deps.PermissionLoader, deps.IdempotencyStore)
	}

	// Role routes: Service Key + Admin JWT
	roleGroup := protected.Group("")
	roleGroup.Use(middleware.JWTAuth(deps.JWTService))
	RegisterRoleRoutes(roleGroup, deps.RoleHandler, deps.PermissionLoader)

	return r
}

// configureTrustedProxies applies the CIDR list; on parse error falls back to
// no trusted proxies (safer than the Gin default of trusting all).
func configureTrustedProxies(r *gin.Engine, raw string) {
	cidrs := splitAndTrim(raw)
	if setErr := r.SetTrustedProxies(cidrs); setErr != nil {
		slog.Warn("invalid HTTP_TRUSTED_PROXIES, falling back to no trusted proxies",
			"error", setErr.Error(), "value", raw)
		_ = r.SetTrustedProxies(nil)
	}
}

// splitAndTrim splits raw by comma and drops empty entries.
func splitAndTrim(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// registerSwaggerRoutes registra rotas do Swagger
func registerSwaggerRoutes(r *gin.Engine) {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

// registerHealthRoutes registra rotas de health check
func registerHealthRoutes(r *gin.Engine, deps Dependencies) {
	// Liveness - always ok (K8s restart if process is dead)
	r.GET("/health", func(c *gin.Context) {
		httpgin.SendSuccess(c, http.StatusOK, gin.H{
			"status":  "ok",
			"service": deps.Config.ServiceName,
		})
	})

	// Readiness - checks all dependencies
	r.GET("/ready", func(c *gin.Context) {
		healthy, statuses := deps.HealthChecker.RunAll(c.Request.Context())

		result := gin.H{
			"status": "ready",
		}
		if !healthy {
			result["status"] = "not ready"
		}
		// Outside dev, hide dependency names from the public probe.
		if deps.Config.Env == "development" {
			result["checks"] = statuses
		}

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}
		httpgin.SendSuccess(c, status, result)
	})
}
