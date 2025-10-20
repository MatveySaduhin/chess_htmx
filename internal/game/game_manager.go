package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/notnil/chess"
)

type GameManager struct {
	games map[string]*Game
	mutex sync.RWMutex
}

type Game struct {
	ID                 string
	chess              *chess.Game
	White              *Player
	Black              *Player
	Status             string
	CreatedAt          time.Time
	FormattedCreatedAt string
	mutex              sync.RWMutex
}

type Player struct {
	UserID string
	Name   string
	Side   chess.Color
	Ready  bool
}

func (g *Game) ChessGame() *chess.Game {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.chess
}

func (gm *GameManager) CanUserAccessGame(userID, gameID string) bool {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return false
	}
	return game.White.UserID == userID || game.Black.UserID == userID
}

func (gm *GameManager) FindAvailableGame() *Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	for _, game := range gm.games {
		// A game is available if it has only one player (white)
		// and black slot is empty
		if game.White != nil && game.Black == nil {
			return game
		}
	}
	return nil
}

func (gm *GameManager) GetPlayerColor(userID, gameID string) chess.Color {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return chess.White // Default fallback
	}
	if game.White.UserID == userID {
		return chess.White
	}
	return chess.Black
}

func NewGameManager() *GameManager {
	return &GameManager{
		games: make(map[string]*Game),
	}
}

func (gm *GameManager) CreateGame(userID, userName string) *Game {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game := &Game{
		ID:        gm.GenerateUniqueGameID(),
		chess:     chess.NewGame(),
		Status:    "waiting",
		CreatedAt: time.Now(),
		White: &Player{
			UserID: userID,
			Name:   userName,
			Side:   chess.White,
			Ready:  true,
		},
	}

	gm.games[game.ID] = game
	return game
}

func (gm *GameManager) JoinGame(gameID, userID, userName string) error {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game, exists := gm.games[gameID]
	if !exists {
		return fmt.Errorf("game not found")
	}

	game.mutex.Lock()
	defer game.mutex.Unlock()

	if game.Status != "waiting" {
		return fmt.Errorf("game already started")
	}

	if game.Black != nil {
		return fmt.Errorf("game already full")
	}

	// Add second player
	game.Black = &Player{
		UserID: userID,
		Name:   userName,
		Side:   chess.Black,
		Ready:  true,
	}

	// Start the game if both players are ready
	if game.White.Ready && game.Black.Ready {
		game.Status = "active"
	}

	return nil
}

func (gm *GameManager) CleanupOldGames() {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	for id, game := range gm.games {
		// Remove games older than 1 hour with only one player
		if game.Black == nil && time.Since(game.CreatedAt) > time.Hour {
			delete(gm.games, id)
		}
	}
}

func (gm *GameManager) GetActiveGames() []*Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	var active []*Game
	for _, game := range gm.games {
		if game.Status == "active" && game.White != nil && game.Black != nil {
			active = append(active, game)
		}
	}
	return active
}

func (gm *GameManager) GetGameStats() map[string]int {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	stats := map[string]int{
		"total":    len(gm.games),
		"active":   0,
		"waiting":  0,
		"finished": 0,
	}

	for _, game := range gm.games {
		switch game.Status {
		case "active":
			stats["active"]++
		case "waiting":
			stats["waiting"]++
		case "finished":
			stats["finished"]++
		}
	}
	return stats
}

func (gm *GameManager) GetWaitingGames() []*Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	var waiting []*Game
	for _, game := range gm.games {
		if game.Status == "waiting" && game.Black == nil {
			waiting = append(waiting, game)
		}
	}
	return waiting
}

func (gm *GameManager) GetUserGames(userID string) []*Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	var userGames []*Game
	for _, game := range gm.games {
		if (game.White != nil && game.White.UserID == userID) ||
			(game.Black != nil && game.Black.UserID == userID) {
			userGames = append(userGames, game)
		}
	}
	return userGames
}

func (gm *GameManager) GetGame(id string) *Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return gm.games[id]
}

func generateGameID() string {
	bytes := make([]byte, 3) // 6 characters in hex
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (gm *GameManager) GenerateUniqueGameID() string {
	for {
		id := generateGameID()
		if _, exists := gm.games[id]; !exists {
			return id
		}
	}
}
