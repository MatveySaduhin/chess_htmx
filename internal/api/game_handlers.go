package api

import (
    "fmt"
    "strings"
    "net/http"
    "github.com/gin-gonic/gin"
)

func (h *Server) GamePage(c *gin.Context) {
    gameID := c.Param("id")
    userID := c.GetString("user_id")

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

func (h *Server) GameWebsocket(c *gin.Context) {
    gameID := c.Param("id")
    userID, exists := c.Get("user-id")
    if !exists {
        c.JSON(401, gin.H{"error": "Not authenticated"})
        return
    }

    if h.gameManager.CanUserAccessGame(userID.(string), gameID) {
        c.JSON(403, gin.H{"error": "Access denied"})
        return
    }

    h.websocketHub.ServeGameWebSocket(c, gameID, userID.(string))
}

func (h *Server) CreateGame(c *gin.Context) {
    userID, exists := c.Get("user_id")
    if !exists {
        c.JSON(401, gin.H{"error": "Not authenticated"})
        return
    }
    userName, _ := c.Get("user_name")

    // Create game with user info
    game := h.gameManager.CreateGame(userID.(string), userName.(string))

    c.Header("HX-Redirect", fmt.Sprintf("/game/%s", game.ID))
    c.JSON(http.StatusOK, gin.H{
        "GameID": game.ID,
        "Color":  "white", 
    })
}

func (h *Server) JoinGame(c *gin.Context) {
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

    userName, _ := c.Get("user_name")

    err := h.gameManager.JoinGame(gameID, userID.(string), userName.(string))
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.Header("HX-Redirect", fmt.Sprintf("game/%s", gameID))
    c.JSON(http.StatusOK, gin.H{
        "GameID": gameID,
        "Color":  "black",
    })
}
