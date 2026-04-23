package middleware

import (
	"context"
	"net/http"
	"slices"

	"github.com/DenysonJ/financial-wallet/pkg/httputil/httpgin"
	"github.com/DenysonJ/financial-wallet/pkg/logutil"
	"github.com/gin-gonic/gin"
)

const ContextKeyPermissions = "user_permissions"
const ContextKeyRoles = "user_roles"

// PermissionLoader loads permissions and roles for a given user ID.
type PermissionLoader interface {
	GetPermissions(ctx context.Context, userID string) ([]string, error)
	GetRoles(ctx context.Context, userID string) ([]string, error)
}

// RequirePermission returns a middleware that checks if the authenticated user
// has the required permission.
//
// Behavior:
//   - If no user_id in context: error 401
//   - If user_id present: load permissions via PermissionLoader and check
//   - Returns 403 if the user lacks the required permission
func RequirePermission(loader PermissionLoader, requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			logutil.LogError(c.Request.Context(), "user not found in context")
			httpgin.SendError(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		userIDStr, ok := userID.(string)
		if !ok || userIDStr == "" {
			logutil.LogError(c.Request.Context(), "user not found in context")
			httpgin.SendError(c, http.StatusUnauthorized, "user not authenticated")
			c.Abort()
			return
		}

		permissions, loadErr := loader.GetPermissions(c.Request.Context(), userIDStr)
		if loadErr != nil {
			logutil.LogError(c.Request.Context(), "failed to load permissions",
				"user.id", userIDStr, "error", loadErr.Error())
			httpgin.SendError(c, http.StatusInternalServerError, "internal server error")
			c.Abort()
			return
		}

		if !slices.Contains(permissions, requiredPermission) {
			logutil.LogWarn(c.Request.Context(), "auth rejected",
				"reason", "missing_permission",
				"user.id", userIDStr,
				"required", requiredPermission)
			httpgin.SendError(c, http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}

		// Load roles for downstream use (e.g., admin checks)
		roles, rolesErr := loader.GetRoles(c.Request.Context(), userIDStr)
		if rolesErr != nil {
			logutil.LogError(c.Request.Context(), "failed to load roles",
				"user.id", userIDStr, "error", rolesErr.Error())
			httpgin.SendError(c, http.StatusInternalServerError, "internal server error")
			c.Abort()
			return
		}

		// Store permissions and roles in context for downstream use
		c.Set(ContextKeyPermissions, permissions)
		c.Set(ContextKeyRoles, roles)
		c.Next()
	}
}
