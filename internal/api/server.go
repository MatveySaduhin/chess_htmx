package api

import (
    "chess_htmx/internal/game"
    "chess_htmx/internal/wsmanager"
)

type Server struct {
    gameManager    *game.GameManager
    authService    *AuthService
    sessionService *SessionService
    websocketHub   *wsmanager.GameHub
}

func NewServer(gm *game.GameManager, as *AuthService, ss *SessionService, wh *wsmanager.GameHub) *Server {
    return &Server{
        authService: as,
        gameManager: gm,
        sessionService: ss,
        websocketHub: wh,
    }
}

