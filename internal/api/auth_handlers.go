package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LoginRequest struct {
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required,min=6"`
}

type RegistrationRequest struct {
	Name     string `form:"name"`
	Email    string `form:"email" binding:"required,email"`
	Password string `form:"password" binding:"required,min=6"`
}

func (s *Server) AuthPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth.html", gin.H{
		"Title": "Authentication",
	})
}

func (s *Server) RegPage(c *gin.Context) {
	c.HTML(http.StatusOK, "regpage.html", gin.H{
		"Title": "Registration",
	})
}

// API:
func (s *Server) Login(c *gin.Context) {
	if s.authService == nil {
		c.JSON(500, gin.H{"error": "Service not available"})
		return
	}

	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	user, err := s.authService.Authenticate(req.Email, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		log.Print(err)
		return
	}

	session, err := s.sessionService.CreateSession(user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create session"})
		return
	}

	c.Header("HX-Redirect", "/")
	c.SetCookie("session_token", session.Token, 3600*24, "/", "", false, true)
	c.Status(http.StatusOK)
	c.JSON(200, gin.H{
		"message": fmt.Sprintf("Welcome back, %s!", user.Name),
		"session": "fake-session-12345",
	})
}

func (s *Server) Logout(c *gin.Context) {
	token, err := c.Cookie("session_token")
	if err == nil {
		s.sessionService.DeleteSession(token)
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.Header("HX-Redirect", "/")
	c.Status(http.StatusOK)
}

func (s *Server) Register(c *gin.Context) {
	name := c.PostForm("name")
	email := c.PostForm("email")
	password := c.PostForm("password")

	log.Printf("DEBUG Registration - name: %s, email: %s", name, email)
	if name == "" || email == "" || password == "" {
		c.String(http.StatusBadRequest, "<div class='text-red-500'>All fields are required</div>")
		return
	}

	user, err := s.authService.NewUser(name, email, password, "user")
	if err != nil {
		log.Printf("DEBUG - Registration failed: %v", err)
		c.String(http.StatusBadRequest, fmt.Sprintf("<div class='text-red-500'>Registration failed: %v</div>", err))
		return
	}
	log.Printf("DEBUG - Registration successful for user: %s, redirecting to /auth", user.Email)
	// Tell htmx to redirect to the login page
	c.Header("HX-Redirect", "/auth")
	c.String(http.StatusCreated, "<div class='text-green-500'>Account created! Please log in.</div>")
}

func (s *Server) newTemplateData(c *gin.Context) gin.H {
	data := gin.H{}
	if user, exists := c.Get("user"); exists {
		data["User"] = user
	}
	return data
}
