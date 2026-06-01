package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")
		if err != nil {
			c.Redirect(302, "/auth")
			c.Abort()
			return
		}

		session, err := s.sessionService.ValidateSession(token)
		if err != nil {
			// Clear invalid cookie
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(302, "/auth")
			c.Abort()
			return
		}

		user, err := s.authService.GetUserByID(session.UserID)
		if err != nil {
			c.SetCookie("session_token", "", -1, "/", "", false, true)
			c.Redirect(302, "/auth")
			c.Abort()
			return
		}

		// Store user ID in context for handlers to use
		// NOTE: .Hex() converts primitve.ObjectID to string
		c.Set("user_id", session.UserID.Hex())
		c.Set("user_name", user.Name)
		c.Set("user", user)
		c.Next()
	}
}

func (s *Server) EasyAuthJWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(authHeader, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing_bearer_token"})
			return
		}

		claims, err := s.easyAuthJWTValidator.Validate(c.Request.Context(), strings.TrimSpace(token))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid_bearer_token"})
			return
		}

		c.Set("auth_user_id", claims.Subject)
		c.Set("auth_session_id", claims.SessionID)
		c.Set("user_id", claims.Subject)
		c.Next()
	}
}

// AdminMiddleware checks if the user in the context has the 'admin' role.
// This middleware MUST be placed *after* the AuthMiddleware in the chain.
func (s *Server) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, exists := c.Get("user")
		if !exists {
			// This should not happen if AuthMiddleware is working correctly
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden. User not found in context."})
			return
		}

		user, ok := userCtx.(*User)
		if !ok || user.Role != "admin" {
			// The user is logged in but is not an admin
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden. Admin access required."})
			return
		}

		c.Next()
	}
}
