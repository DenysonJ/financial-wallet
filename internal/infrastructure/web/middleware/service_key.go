package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
)

const ContextKeyServiceKey = "serviceKey"

// ServiceKeyConfig contém a configuração de autenticação por Service Key.
type ServiceKeyConfig struct {
	// Enabled indica se a autenticação está habilitada.
	// Quando true e Keys está vazio, rejeita todas as requisições com 503 (fail-closed).
	// Quando false, permite todas as requisições (modo desenvolvimento).
	Enabled bool

	// Keys é um mapa de service-name → key autorizado.
	// Formato de entrada (env): "service1:key1,service2:key2"
	Keys map[string]string

	// ServiceNameHeader é o header que contém o nome do serviço chamador.
	// Default: "Service-Name"
	ServiceNameHeader string

	// ServiceKeyHeader é o header que contém a chave do serviço.
	// Default: "Service-Key"
	ServiceKeyHeader string
}

// ParseServiceKeys converte uma string no formato "service1:key1,service2:key2"
// para um mapa de chaves.
func ParseServiceKeys(raw string) map[string]string {
	keys := make(map[string]string)
	if raw == "" {
		return keys
	}

	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			serviceName := strings.TrimSpace(parts[0])
			serviceKey := strings.TrimSpace(parts[1])
			if serviceName != "" && serviceKey != "" {
				keys[serviceName] = serviceKey
			}
		}
	}
	return keys
}

// DefaultServiceKeyConfig retorna a configuração padrão.
func DefaultServiceKeyConfig() ServiceKeyConfig {
	return ServiceKeyConfig{
		Keys:              make(map[string]string),
		ServiceNameHeader: "Service-Name",
		ServiceKeyHeader:  "Service-Key",
	}
}

// ServiceKeyAuth retorna um middleware que valida a autenticação via Service Key.
//
// Comportamento baseado no campo Enabled:
//   - Enabled=false: permite todas as requisições (modo desenvolvimento)
//   - Enabled=true + Keys vazio: rejeita todas as requisições com 503 (fail-closed)
//   - Enabled=true + Keys populado: valida normalmente
func ServiceKeyAuth(config ServiceKeyConfig) gin.HandlerFunc {
	// Define headers padrão se não configurados
	if config.ServiceNameHeader == "" {
		config.ServiceNameHeader = "Service-Name"
	}
	if config.ServiceKeyHeader == "" {
		config.ServiceKeyHeader = "Service-Key"
	}

	return func(c *gin.Context) {
		// Se auth não está habilitada, permite tudo (modo desenvolvimento)
		if !config.Enabled {
			c.Next()
			return
		}

		// Fail-closed: auth habilitada mas sem chaves configuradas.
		// Impede que um deploy sem SERVICE_KEYS em HML/PRD exponha o serviço.
		if len(config.Keys) == 0 {
			logutil.LogWarn(c.Request.Context(), "auth rejected", "reason", "service_keys_not_configured")
			httpgin.SendError(c, http.StatusServiceUnavailable, "service authentication not configured")
			c.Abort()
			return
		}

		serviceName := c.GetHeader(config.ServiceNameHeader)
		serviceKey := c.GetHeader(config.ServiceKeyHeader)

		// Validate headers present
		if serviceName == "" || serviceKey == "" {
			logutil.LogWarn(c.Request.Context(), "auth rejected", "reason", "missing_service_headers")
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Validate service key — always run ConstantTimeCompare, even when the
		// service name is unknown, so the response time does not reveal whether
		// a given service name is configured. Without the dummy compare, an
		// attacker can enumerate valid service names via timing.
		expectedKey, exists := config.Keys[serviceName]
		if !exists {
			// Compare against a fixed dummy of the same shape as a real key so
			// the code path takes comparable time before we return 401.
			_ = subtle.ConstantTimeCompare([]byte("00000000000000000000000000000000"), []byte(serviceKey))
			logutil.LogWarn(c.Request.Context(), "auth rejected", "reason", "unknown_service", "service", serviceName)
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		if subtle.ConstantTimeCompare([]byte(expectedKey), []byte(serviceKey)) != 1 {
			logutil.LogWarn(c.Request.Context(), "auth rejected", "reason", "invalid_service_key", "service", serviceName)
			httpgin.SendError(c, http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		// Enrich LogContext with caller service for downstream logging
		lc, _ := logutil.Extract(c.Request.Context())
		lc.CallerService = serviceName
		c.Request = c.Request.WithContext(logutil.Inject(c.Request.Context(), lc))
		c.Set(ContextKeyServiceKey, serviceName)

		c.Next()
	}
}
