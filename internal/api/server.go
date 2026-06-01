package api

import (
	"chess_htmx/internal/game"
	"chess_htmx/internal/wsmanager"
	"os"
)

type Server struct {
	gameManager          *game.GameManager
	authService          *AuthService
	sessionService       *SessionService
	profileService       *ProfileService
	easyAuthClient       *EasyAuthClient
	easyAuthJWTValidator *EasyAuthJWTValidator
	websocketHub         *wsmanager.GameHub
	engineService        *EngineService
}

func NewServer(gm *game.GameManager, as *AuthService, ss *SessionService, ps *ProfileService, eac *EasyAuthClient, validator *EasyAuthJWTValidator, wh *wsmanager.GameHub) *Server {
	stockfishPath := os.Getenv("STOCKFISH_PATH")
	if stockfishPath == "" {
		stockfishPath = "stockfish"
	}

	return &Server{
		authService:          as,
		gameManager:          gm,
		sessionService:       ss,
		profileService:       ps,
		easyAuthClient:       eac,
		easyAuthJWTValidator: validator,
		websocketHub:         wh,
		engineService:        NewEngineService(stockfishPath),
	}
}
