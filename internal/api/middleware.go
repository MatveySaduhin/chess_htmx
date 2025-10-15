package api

import 	"github.com/gin-gonic/gin"

func (h *Server) AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token, err := c.Cookie("session_token")
        if err != nil {
            c.Redirect(302, "/auth")
            c.Abort()
            return
        }

        session, err := h.sessionService.ValidateSession(token)
        if err != nil {
            // Clear invalid cookie
            c.SetCookie("session_token", "", -1, "/", "", false, true)
            c.Redirect(302, "/auth")
            c.Abort()
            return
        }

        user, err := h.authService.GetUserByID(session.UserID)
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
        c.Next()
    }
}
