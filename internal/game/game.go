package game

import (
    "fmt"
    "sync"
    "github.com/notnil/chess"
    "crypto/rand"
    "encoding/hex"
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
    UserID   string
    Name     string
    Side     chess.Color
    Ready    bool
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

func (gm *GameManager) CreateGame(userID, userName string) *Game {
    gm.mutex.Lock()
    defer gm.mutex.Unlock()

    game := &Game{
        ID:     gm.generateUniqueGameID(),
        chess:  chess.NewGame(),
        Status: "waiting",
        White: &Player{
            UserID: userID,
            Name: userName,
            Side: chess.White,
            Ready: true,
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

func (gm *GameManager) generateUniqueGameID() string {
    for {
        id := generateGameID()
        if _, exists := gm.games[id]; !exists {
            return id
        }
    }
}
