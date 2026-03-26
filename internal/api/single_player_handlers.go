package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type SinglePlayerMoveRequest struct {
	GameID string `json:"game_id" binding:"required"`
	Move   string `json:"move" binding:"required"`
}

type UndoMoveRequest struct {
	GameID string `json:"game_id" binding:"required"`
}

func (s *Server) SinglePlayerMove(c *gin.Context) {
	userID := c.GetString("user_id")

	var req SinglePlayerMoveRequest
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

	fen, err := s.gameManager.ApplySANMove(req.GameID, req.Move)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid move: " + err.Error()})
		return
	}

	opening := lookupOpeningBySAN(gameObj.MoveHistorySAN())
	
	c.JSON(http.StatusOK, gin.H{
		"fen":  fen,
		"move": req.Move,
		"opening": gin.H{
			"eco":   opening.ECO,
			"name":  opening.Name,
			"moves": opening.Moves,
		},
		"mode":   string(gameObj.Mode),
		"status": gameObj.Status,
	})
}

func (s *Server) UndoSinglePlayerMove(c *gin.Context) {
	userID := c.GetString("user_id")

	var req UndoMoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if !s.gameManager.CanUserAccessGame(userID, req.GameID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	fen, err := s.gameManager.UndoLastMove(req.GameID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Undo failed: " + err.Error()})
		return
	}

	opening := lookupOpeningBySAN(s.gameManager.GetGame(req.GameID).MoveHistorySAN())

	c.JSON(http.StatusOK, gin.H{
		"fen": fen,
		"opening": gin.H{
			"eco":   opening.ECO,
			"name":  opening.Name,
			"moves": opening.Moves,
		},
	})
}
