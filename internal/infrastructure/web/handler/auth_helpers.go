package handler

import (
	"slices"

	"github.com/DenysonJ/financial-wallet/internal/infrastructure/web/middleware"
	"github.com/gin-gonic/gin"
)

// isServiceKeyRequest returns true if the request was authenticated via Service Key.
func isServiceKeyRequest(c *gin.Context) bool {
	_, exists := c.Get(middleware.ContextKeyServiceKey)
	return exists
}

// isAdmin checks if the JWT user has the "admin" role.
func isAdmin(c *gin.Context) bool {
	roles, exists := c.Get("user_roles")
	if !exists {
		return false
	}
	if roleSlice, ok := roles.([]string); ok {
		return slices.Contains(roleSlice, "admin")
	}
	return false
}

// isAdminOrOwner checks if the JWT user is the resource owner or has admin-level permissions.
// Returns true for Service Key requests (trusted), admin users, or matching owner.
func isAdminOrOwner(c *gin.Context, resourceUserID string) bool {
	if isServiceKeyRequest(c) {
		return true
	}

	jwtUserID, _ := c.Get(middleware.ContextKeyUserID)
	jwtUserIDStr, _ := jwtUserID.(string)

	if jwtUserIDStr == resourceUserID {
		return true
	}

	return isAdmin(c)
}

// getRequiredJWTUserID extracts the user ID from JWT context.
// Returns the user ID and true if present, or empty string and false if missing.
func getRequiredJWTUserID(c *gin.Context) (userID string, ok bool) {
	raw, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return "", false
	}
	userIDStr, _ := raw.(string)
	if userIDStr == "" {
		return "", false
	}
	return userIDStr, true
}

// ownershipUserID returns the user ID for ownership enforcement.
// Admin and service-key requests return "" (skip check); regular users return their ID.
func ownershipUserID(c *gin.Context) string {
	if isServiceKeyRequest(c) || isAdmin(c) {
		return ""
	}
	userID, _ := getRequiredJWTUserID(c)
	return userID
}
