package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/notnil/chess"
)

type EngineMoveRequest struct {
	GameID string `json:"game_id" binding:"required"`
	Level  string `json:"level"`
}

func uciToChessMove(position *chess.Position, uci string) (*chess.Move, error) {
	return chess.UCINotation{}.Decode(position, uci)
}

func (s *Server) EngineMove(c *gin.Context) {
	userID := c.GetString("user_id")

	var req EngineMoveRequest
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

	if string(gameObj.Mode) != "vs_computer" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Engine move only available in vs_computer mode"})
		return
	}

	level := EngineLevel(req.Level)
	if level == "" {
		level = EngineMedium
	}

	bestMoveUCI, err := s.engineService.BestMove(gameObj.FEN(), level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Engine failed: " + err.Error()})
		return
	}

	engineMove, err := uciToChessMove(gameObj.ChessGame().Position(), bestMoveUCI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Engine move decode failed: " + err.Error()})
		return
	}

	san := chess.AlgebraicNotation{}.Encode(gameObj.ChessGame().Position(), engineMove)

	fen, err := s.gameManager.ApplySANMove(req.GameID, san)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Applying engine move failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fen":          fen,
		"engine_move":  san,
		"engine_uci":   bestMoveUCI,
		"engine_level": string(level),
	})
}
