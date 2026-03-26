package api

import (
	"chess_htmx/internal/game"
	"chess_htmx/internal/wsmanager"
	"os"
)

type Server struct {
	gameManager    *game.GameManager
	authService    *AuthService
	sessionService *SessionService
	websocketHub   *wsmanager.GameHub
	engineService  *EngineService
}

func NewServer(gm *game.GameManager, as *AuthService, ss *SessionService, wh *wsmanager.GameHub) *Server {
	stockfishPath := os.Getenv("STOCKFISH_PATH")
	if stockfishPath == "" {
		stockfishPath = "stockfish"
	}

	return &Server{
		authService:    as,
		gameManager:    gm,
		sessionService: ss,
		websocketHub:   wh,
		engineService:  NewEngineService(stockfishPath),
	}
}
