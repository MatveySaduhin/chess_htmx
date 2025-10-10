package api

import (
    "net/http"
    "chess_htmx/internal/game"
    "github.com/gin-gonic/gin"
)

type Handlers struct {
    gameManager *game.GameManager
    authService *AuthService
}

func NewHandlers(gm *game.GameManager) *Handlers {
    return &Handlers{
        authService: &AuthService{},
        gameManager: gm,
    //  authServeice not being defined yet
    }
}

func (h *Handlers) AuthPage(c *gin.Context) {
    c.HTML(http.StatusOK, "auth.html", gin.H{
        "Title": "Authentication",
    })
}

type LoginRequest struct {
    Email    string `form:"email" binding:"required,email"`
    Password string `form:"password" binding:"required,min=6"`
}

func (h *Handlers) Login(c *gin.Context) {
    var req LoginRequest

    // Bind form data to struct
    if err := c.ShouldBind(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Now use req.Email and req.Password
    err := h.authService.Authenticate(req.Email, req.Password)
    if err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    c.JSON(200, gin.H{"message": "Login successful", "user": req.Email})
}

func (h *Handlers) Home(c *gin.Context) {
    c.HTML(http.StatusOK, "home.html", gin.H{
        "Title": "Chess App",
    })
}
/*
func (h *Handlers) Game(c *gin.Context) {
    c.HTML(http.StatusOK, "game.html", gin.H{
        "GameID": 1,
    })
}
*/
func (h *Handlers) CreateGame(c *gin.Context) {
    game := h.gameManager.CreateGame()

    c.HTML(http.StatusOK, "game_created.html", gin.H{
        "GameID": game.ID,
    })
}
