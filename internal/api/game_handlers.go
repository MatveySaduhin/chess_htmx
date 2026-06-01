package api

import (
	"chess_htmx/internal/game"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/notnil/chess"
)

type CreateSinglePlayerRequest struct {
	Mode      string `form:"mode" binding:"required"`
	UserColor string `form:"user_color"`
}

type SurrenderRequest struct {
	GameID string `json:"game_id" binding:"required"`
}

func (s *Server) GamePage(c *gin.Context) {
	gameID := c.Param("id")
	userID := c.GetString("user_id")
	currentUserName, _ := c.Get("user_name")

	if !s.gameManager.CanUserAccessGame(userID, gameID) {
		c.Redirect(http.StatusFound, "/")
		return
	}

	color := s.gameManager.GetPlayerColor(userID, gameID)
	gameObj := s.gameManager.GetGame(gameID)
	if gameObj == nil {
		c.Redirect(http.StatusFound, "/")
		return
	}

	var whitePlayerName, blackPlayerName string

	if gameObj.White != nil {
		whitePlayerName = gameObj.White.Name
	}
	if gameObj.Black != nil {
		blackPlayerName = gameObj.Black.Name
	}

	c.HTML(http.StatusOK, "game.html", gin.H{
		"GameID":          gameID,
		"Color":           strings.ToLower(color.Name()),
		"CurrentUserName": currentUserName,
		"WhitePlayerName": whitePlayerName,
		"BlackPlayerName": blackPlayerName,
		"IsWhite":         color == chess.White,
		"IsBlack":         color == chess.Black,
		"Mode":            string(gameObj.Mode),
		"IsSinglePlayer":  gameObj.Mode != game.ModeMultiplayer,
		"IsOpeningStudy":  gameObj.Mode == game.ModeOpeningStudy,
		"IsVsComputer":    gameObj.Mode == game.ModeVsComputer,
		"InitialFEN":      gameObj.FEN(),
		"IsEasyAuth":      false,
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

	availableGame := s.gameManager.FindAvailableGame()
	if availableGame != nil {
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
		return
	}

	gameObj := s.gameManager.CreateGame(userID.(string), userName.(string))
	c.Header("HX-Redirect", fmt.Sprintf("/game/%s", gameObj.ID))
	c.JSON(http.StatusOK, gin.H{
		"GameID":  gameObj.ID,
		"Color":   "white",
		"message": "Created new game - waiting for opponent",
	})
}

func (s *Server) CreateGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return
	}
	userName, _ := c.Get("user_name")

	gameObj := s.gameManager.CreateGame(userID.(string), userName.(string))

	c.Header("HX-Redirect", fmt.Sprintf("/game/%s", gameObj.ID))
	c.JSON(http.StatusOK, gin.H{
		"GameID": gameObj.ID,
		"Color":  "white",
	})
}

func (s *Server) JoinGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "Not authenticated"})
		return
	}

	gameID := c.PostForm("game_id")
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

func (s *Server) CreateSinglePlayerGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	userName, _ := c.Get("user_name")

	var req CreateSinglePlayerRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form: " + err.Error()})
		return
	}

	switch req.Mode {
	case string(game.ModeVsComputer):
		userColor := chess.White
		if strings.ToLower(req.UserColor) == "black" {
			userColor = chess.Black
		}

		gameObj := s.gameManager.CreateVsComputerGame(userID.(string), userName.(string), userColor)
		c.Header("HX-Redirect", fmt.Sprintf("/game/%s", gameObj.ID))
		c.JSON(http.StatusOK, gin.H{
			"GameID": gameObj.ID,
			"Mode":   req.Mode,
		})
		return

	case string(game.ModeOpeningStudy):
		gameObj := s.gameManager.CreateOpeningStudyGame(userID.(string), userName.(string))
		c.Header("HX-Redirect", fmt.Sprintf("/game/%s", gameObj.ID))
		c.JSON(http.StatusOK, gin.H{
			"GameID": gameObj.ID,
			"Mode":   req.Mode,
		})
		return

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown single player mode"})
	}
}

func (s *Server) SurrenderGame(c *gin.Context) {
	userID := c.GetString("user_id")

	var req SurrenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if !s.gameManager.CanUserAccessGame(userID, req.GameID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	gameObj := s.gameManager.GetGame(req.GameID)
	if gameObj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	winner, result, fen, err := s.gameManager.SurrenderGame(req.GameID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id": req.GameID,
		"winner":  winner,
		"result":  result,
		"fen":     fen,
		"mode":    string(gameObj.Mode),
		"status":  "finished",
	})
}
