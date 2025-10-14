package api

import (
    "strings"
    "net/http"
    "github.com/gin-gonic/gin"
)

func (h *Handlers) GamePage(c *gin.Context) {
    gameID := c.Param("id")
    userID := c.GetString("user_id") // From auth middleware

    if !h.gameManager.CanUserAccessGame(userID, gameID) {
        c.Redirect(http.StatusFound, "/")
        return
    }

    color := h.gameManager.GetPlayerColor(userID, gameID)

    c.HTML(http.StatusOK, "game.html", gin.H{
        "GameID": gameID,
        "Color":  strings.ToLower(color.Name()),
    })
}
