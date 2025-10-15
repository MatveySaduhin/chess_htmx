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

// Pages: 
// (Home probably don't belong here, creating separate file feels wrong)
//  TODO: Implement feature rich Home screen so I can feel free to create separate file for it
func (h *Server) Home(c *gin.Context) {
    var userName string
    token, err := c.Cookie("session_token")
    if err == nil {
        session, err := h.sessionService.ValidateSession(token)
        if err == nil {
            user, err := h.authService.GetUserByID(session.UserID)
            if err == nil {
                userName = user.Name
            }
        }
    }

    c.HTML(http.StatusOK, "home.html", gin.H{
        "Title":    "Chess App",
        "UserName": userName,
    })
}

func (h *Server) AuthPage(c *gin.Context) {
    c.HTML(http.StatusOK, "auth.html", gin.H{
        "Title": "Authentication",
    })
}

func (h *Server) RegPage(c *gin.Context) {
    c.HTML(http.StatusOK, "regpage.html", gin.H{
        "Title": "Registration",
    })
}

// API:
//
func (h *Server) Login(c *gin.Context) {
    if h.authService == nil {
        c.JSON(500, gin.H{"error": "Service not available"})
        return
    }

    var req LoginRequest
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    user, err := h.authService.Authenticate(req.Email, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": err.Error()})
        log.Print(err)
        return
    }

    session, err := h.sessionService.CreateSession(user.ID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create session"})
        return
    }

    c.SetCookie("session_token", session.Token, 3600*24, "/", "", false, true)

    c.Header("HX-Redirect", "/")
    c.JSON(200, gin.H{
        "message": fmt.Sprintf("Welcome back, %s!", user.Name),
        "session": "fake-session-12345",
    })
}

func (h *Server) Logout(c *gin.Context) {
    token, err := c.Cookie("session_token")
    if err == nil {
        h.sessionService.DeleteSession(token)
    }

    c.SetCookie("session_token", "", -1, "/", "", false, true)
    c.Header("HX-Redirect", "/")
    c.JSON(200, gin.H{"message": "Logged out successfully"})
}

/*
 BUG:  Neither registers, nor throws an error.
       Most likely related to db.
 NOTE: Included login after registration,
       so the function actually throws 401 on login stage.
*/
func (h *Server) Register(c *gin.Context) {
    if h.authService == nil {
        c.JSON(500, gin.H{"error": "Service not available"})
        return
    }

    var req RegistrationRequest
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    user, err := h.authService.NewUser(req.Name, req.Email, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": err.Error()})
        return
    }

    user, err = h.authService.Authenticate(req.Email, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": err.Error()})
  // WARNING:
        log.Print(err)
        return
    }

    session, err := h.sessionService.CreateSession(user.ID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create session"})
        return
    }

    c.SetCookie("session_token", session.Token, 3600*24, "/", "", false, true)

    c.Header("HX-Redirect", "/")
    c.JSON(200, gin.H{
        "message": fmt.Sprintf("Welcome back, %s!", user.Name),
        "session": "fake-session-12345",
    })
}
