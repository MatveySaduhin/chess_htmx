package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) GetOpeningInfo(c *gin.Context) {
	gameID := c.Query("game_id")
	if gameID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game_id is required"})
		return
	}

	userID := c.GetString("user_id")
	if !s.gameManager.CanUserAccessGame(userID, gameID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	gameObj := s.gameManager.GetGame(gameID)
	if gameObj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Game not found"})
		return
	}

	opening := lookupOpeningBySAN(gameObj.MoveHistorySAN())

	c.JSON(http.StatusOK, gin.H{
		"opening": gin.H{
			"eco":   opening.ECO,
			"name":  opening.Name,
			"moves": opening.Moves,
		},
		"fen": gameObj.FEN(),
	})
}
