package api

import (
	"chess_htmx/internal/game"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
    gameManager *game.GameManager
    authService *AuthService
    sessionService *SessionService
}

type LoginRequest struct {
    Email    string `form:"email" binding:"required,email"`
    Password string `form:"password" binding:"required,min=6"`
}

type RegistrationRequest struct {
    Name     string `form:"name"`
    Email    string `form:"email" binding:"required,email"`
    Password string `form:"password" binding:"required,min=6"`
}

func NewHandlers(gm *game.GameManager, as *AuthService, ss *SessionService) *Handlers {
    return &Handlers{
        authService: as,
        gameManager: gm,
        sessionService: ss,
    }
}

func (h *Handlers) AuthPage(c *gin.Context) {
    c.HTML(http.StatusOK, "auth.html", gin.H{
        "Title": "Authentication",
    })
}

func (h *Handlers) Login(c *gin.Context) {
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

func (h *Handlers) Logout(c *gin.Context) {
    token, err := c.Cookie("session_token")
    if err == nil {
        h.sessionService.DeleteSession(token)
    }
    
    c.SetCookie("session_token", "", -1, "/", "", false, true)
    c.Header("HX-Redirect", "/")
    c.JSON(200, gin.H{"message": "Logged out successfully"})
}

func (h *Handlers) Register(c *gin.Context) {
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
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(201, gin.H{"message": "User created successfully", "user": user})
}

func (h *Handlers) Home(c *gin.Context) {
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

func (h *Handlers) CreateGame(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "Not authenticated"})
        return
    }
    userName, _ := c.Get("user_name")

    // Create game with user info
    game := h.gameManager.CreateGame(userID.(string), userName.(string))

    c.HTML(http.StatusOK, "game_created.html", gin.H{
        "GameID": game.ID,
        "Color":  "white", 
    })
}

func (h *Handlers) JoinGame(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "Not authenticated"})
        return
    }

    gameID := c.PostForm("game_id") // Get from form data instead of URL param
    if gameID == "" {
        c.JSON(400, gin.H{"error": "Game ID is required"})
        return
    }

    userName := "Player" // Get from session or DB
    
    err := h.gameManager.JoinGame(gameID, userID.(string), userName)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.HTML(http.StatusOK, "game_joined.html", gin.H{
        "GameID": gameID,
        "Color":  "black",
    })
}

func (h *Handlers) MyGames(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "Not authenticated"})
        return
    }

    games := h.gameManager.GetUserGames(userID.(string))

    c.HTML(http.StatusOK, "my_games.html", gin.H{
        "Games": games,
    })
}
