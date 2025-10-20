package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) Home(c *gin.Context) {
	var user *User
	token, err := c.Cookie("session_token")
	if err == nil {
		session, err := s.sessionService.ValidateSession(token)
		if err == nil {
			user, _ = s.authService.GetUserByID(session.UserID)
		}
	}

	c.HTML(http.StatusOK, "home.html", gin.H{
		"Title": "Chess App",
		"User":  user,
	})
}
