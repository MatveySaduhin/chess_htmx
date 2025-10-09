package api

import (
    "net/http"
    "chess_htmx/internal/game"
    "github.com/gin-gonic/gin"
)

type Handlers struct {
    gameManager *game.GameManager
}

func NewHandlers(gm *game.GameManager) *Handlers {
    return &Handlers{
        gameManager: gm,
    }
}

func (h *Handlers) AuthPage(c *gin.Context) {
    c.HTML(http.StatusOK, "auth.html", gin.H{
        "Title": "Authentication",
    })
}

func (h *Handlers) Login(c *gin.Context) {}

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
