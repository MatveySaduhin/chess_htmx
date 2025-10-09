// internal/game/game.go
package game

import (
    "sync"
    "github.com/notnil/chess"
)

type Game struct {
    ID     string
    chess  *chess.Game
    White  *Player
    Black  *Player
    Status string
    mutex  sync.RWMutex
}

type Player struct {
    ID   string
    Name string
    Side chess.Color
}

type GameManager struct {
    games map[string]*Game
    mutex sync.RWMutex
}

func NewGameManager() *GameManager {
    return &GameManager{
        games: make(map[string]*Game),
    }
}

func (gm *GameManager) CreateGame() *Game {
    gm.mutex.Lock()
    defer gm.mutex.Unlock()

    game := &Game{
        ID:     generateGameID(),
        chess:  chess.NewGame(),
        Status: "waiting",
    }

    gm.games[game.ID] = game
    return game
}

func (gm *GameManager) GetGame(id string) *Game {
    gm.mutex.RLock()
    defer gm.mutex.RUnlock()
    return gm.games[id]
}

func generateGameID() string {
    // Simple ID generation - replace with something better
    return "8080" 
}
