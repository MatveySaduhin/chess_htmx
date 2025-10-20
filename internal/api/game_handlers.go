package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/notnil/chess"
	"net/http"
	"strings"
)

func (s *Server) GamePage(c *gin.Context) {
	gameID := c.Param("id")
	userID := c.GetString("user_id")
	currentUserName, _ := c.Get("user_name")

	if !s.gameManager.CanUserAccessGame(userID, gameID) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	color := s.gameManager.GetPlayerColor(userID, gameID)

	game := s.gameManager.GetGame(gameID)

	var whitePlayerName, blackPlayerName string

	if game != nil {
		if game.White != nil {
			whitePlayerName = game.White.Name
		}
		if game.Black != nil {
			blackPlayerName = game.Black.Name
		}
	}
	c.HTML(http.StatusOK, "game.html", gin.H{
		"GameID":          gameID,
		"Color":           strings.ToLower(color.Name()),
		"CurrentUserName": currentUserName,
		"WhitePlayerName": whitePlayerName,
		"BlackPlayerName": blackPlayerName,
		"IsWhite":         color == chess.White,
		"IsBlack":         color == chess.Black,
	})
}

func (s *Server) GameWebsocket(c *gin.Context) {
	gameID := c.Param("id")
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return
	}

	if !s.gameManager.CanUserAccessGame(userID.(string), gameID) {
		c.JSON(403, gin.H{"error": "Access denied"})
		return
	}

	s.websocketHub.ServeGameWebSocket(c, gameID, userID.(string))
}

func (s *Server) QuickPlay(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return
	}
	userName, _ := c.Get("user_name")

	// 1. Look for available games (games waiting for players)
	availableGame := s.gameManager.FindAvailableGame()

	if availableGame != nil {
		// 2. Join existing game
		err := s.gameManager.JoinGame(availableGame.ID, userID.(string), userName.(string))
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.Header("HX-Redirect", fmt.Sprintf("/game/%s", availableGame.ID))
		c.JSON(http.StatusOK, gin.H{
			"GameID":  availableGame.ID,
			"Color":   "black",
			"message": "Joined existing game",
		})
	} else {
		// 3. Create new game if none available
		game := s.gameManager.CreateGame(userID.(string), userName.(string))

		c.Header("HX-Redirect", fmt.Sprintf("/game/%s", game.ID))
		c.JSON(http.StatusOK, gin.H{
			"GameID":  game.ID,
			"Color":   "white",
			"message": "Created new game - waiting for opponent",
		})
	}
}

func (s *Server) CreateGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return
	}
	userName, _ := c.Get("user_name")

	// Create game with user info
	game := s.gameManager.CreateGame(userID.(string), userName.(string))

	c.Header("HX-Redirect", fmt.Sprintf("/game/%s", game.ID))
	c.JSON(http.StatusOK, gin.H{
		"GameID": game.ID,
		"Color":  "white",
	})
}

func (s *Server) JoinGame(c *gin.Context) {
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

	err := s.gameManager.JoinGame(gameID, userID.(string), userName.(string))
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.Header("HX-Redirect", fmt.Sprintf("/game/%s", gameID))
	c.JSON(http.StatusOK, gin.H{
		"GameID": gameID,
		"Color":  "black",
	})
}
