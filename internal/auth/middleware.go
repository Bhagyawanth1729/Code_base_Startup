package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/forgestock/forgestock/internal/platform/http"
	"github.com/forgestock/forgestock/internal/platform/logger"
)

const (
	UserIDKey   = "user_id"
	EmailKey    = "email"
	RolesKey    = "roles"
	ClaimsKey   = "claims"
)

func AuthMiddleware(jwtManager *JWTManager, log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			http.RespondUnauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.RespondUnauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			if errors.Is(err, ErrExpiredToken) {
				http.RespondUnauthorized(c, "Token expired")
			} else {
				http.RespondUnauthorized(c, "Invalid token")
			}
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Set(RolesKey, claims.Roles)
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func OptionalAuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			c.Next()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)
		c.Set(RolesKey, claims.Roles)
		c.Set(ClaimsKey, claims)
		c.Next()
	}
}

func RequireRoles(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get(RolesKey)
		if !exists {
			http.RespondForbidden(c, "Authentication required")
			c.Abort()
			return
		}

		userRoles, ok := rolesVal.([]string)
		if !ok {
			http.RespondForbidden(c, "Invalid role format")
			c.Abort()
			return
		}

		hasRole := false
		for _, userRole := range userRoles {
			for _, allowedRole := range allowedRoles {
				if userRole == allowedRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			http.RespondForbidden(c, "Insufficient permissions")
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetUserID(c *gin.Context) (string, bool) {
	val, exists := c.Get(UserIDKey)
	if !exists {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}

func GetUserEmail(c *gin.Context) (string, bool) {
	val, exists := c.Get(EmailKey)
	if !exists {
		return "", false
	}
	email, ok := val.(string)
	return email, ok
}

func GetUserRoles(c *gin.Context) ([]string, bool) {
	val, exists := c.Get(RolesKey)
	if !exists {
		return nil, false
	}
	roles, ok := val.([]string)
	return roles, ok
}

func GetClaims(c *gin.Context) (*Claims, bool) {
	val, exists := c.Get(ClaimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := val.(*Claims)
	return claims, ok
}