package router

import (
	"net/http"
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
	JWTEnabled         bool
	RateLimitEnabled   bool
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	RateLimitAuthReqs  int
	RateLimitAuthWin   time.Duration
}

// Dependencies agrupa todas as dependências necessárias para o router
type Dependencies struct {
	HealthChecker    *health.Checker
	UserHandler      *handler.UserHandler
	RoleHandler      *handler.RoleHandler
	AuthHandler      *handler.AuthHandler
	PasswordHandler  *handler.PasswordHandler
	JWTService       interfaces.TokenService
	HTTPMetrics      *telemetry.HTTPMetrics
	IdempotencyStore idempotency.Store
	RateLimitStore   ratelimit.Store
	Config           Config
}

// Setup configura e retorna o router Gin com todos os middlewares e rotas
func Setup(deps Dependencies) *gin.Engine {
	r := gin.New()

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
				Store:  deps.RateLimitStore,
				Limit:  deps.Config.RateLimitAuthReqs,
				Window: deps.Config.RateLimitAuthWin,
			}))
		}
		authGroup.POST("/login", deps.AuthHandler.Login)
		authGroup.POST("/refresh", deps.AuthHandler.Refresh)
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
		RegisterSetPasswordRoute(protected, deps.PasswordHandler)
	}

	// User routes + Change Password: Service Key OR JWT authentication
	if deps.Config.JWTEnabled && deps.JWTService != nil {
		jwtProtected := protected.Group("")
		jwtProtected.Use(middleware.JWTAuth(deps.JWTService))
		RegisterUserRoutes(jwtProtected, deps.UserHandler)
		if deps.PasswordHandler != nil {
			RegisterChangePasswordRoute(jwtProtected, deps.PasswordHandler)
		}
	} else {
		RegisterUserRoutes(protected, deps.UserHandler)
		if deps.PasswordHandler != nil {
			RegisterChangePasswordRoute(protected, deps.PasswordHandler)
		}
	}

	// Role routes: Service Key only (no JWT required per spec REQ-5)
	RegisterRoleRoutes(protected, deps.RoleHandler)

	return r
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
		result["checks"] = statuses

		status := http.StatusOK
		if !healthy {
			status = http.StatusServiceUnavailable
		}
		httpgin.SendSuccess(c, status, result)
	})
}
