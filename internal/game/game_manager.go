package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/notnil/chess"
)

type GameMode string

const (
	ModeMultiplayer  GameMode = "multiplayer"
	ModeVsComputer   GameMode = "vs_computer"
	ModeOpeningStudy GameMode = "opening_study"
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
	Result             string
	Winner             string
	CreatedAt          time.Time
	FormattedCreatedAt string
	Mode               GameMode
	AllowFreeMoves     bool
	SANHistory         []string
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

func (g *Game) FEN() string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.chess.FEN()
}

func (g *Game) PGN() string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.chess.String()
}

func (g *Game) PositionTurn() chess.Color {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.chess.Position().Turn()
}

func (g *Game) MoveHistorySAN() []string {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	history := make([]string, len(g.SANHistory))
	copy(history, g.SANHistory)
	return history
}

func (gm *GameManager) CanUserAccessGame(userID, gameID string) bool {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	game, exists := gm.games[gameID]
	if !exists {
		return false
	}

	if game.White != nil && game.White.UserID == userID {
		return true
	}
	if game.Black != nil && game.Black.UserID == userID {
		return true
	}

	return false
}

func (gm *GameManager) FindAvailableGame() *Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	for _, game := range gm.games {
		if game.Mode == ModeMultiplayer && game.White != nil && game.Black == nil && game.Status == "waiting" {
			return game
		}
	}
	return nil
}

func (gm *GameManager) GetPlayerColor(userID, gameID string) chess.Color {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	game, exists := gm.games[gameID]
	if !exists {
		return chess.White
	}

	if game.White != nil && game.White.UserID == userID {
		return chess.White
	}
	if game.Black != nil && game.Black.UserID == userID {
		return chess.Black
	}

	return chess.White
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
		ID:         gm.GenerateUniqueGameID(),
		chess:      chess.NewGame(),
		Status:     "waiting",
		CreatedAt:  time.Now(),
		Mode:       ModeMultiplayer,
		SANHistory: []string{},
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

func (gm *GameManager) CreateVsComputerGame(userID, userName string, userColor chess.Color) *Game {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game := &Game{
		ID:         gm.GenerateUniqueGameID(),
		chess:      chess.NewGame(),
		Status:     "active",
		CreatedAt:  time.Now(),
		Mode:       ModeVsComputer,
		SANHistory: []string{},
	}

	if userColor == chess.White {
		game.White = &Player{
			UserID: userID,
			Name:   userName,
			Side:   chess.White,
			Ready:  true,
		}
		game.Black = &Player{
			UserID: "computer",
			Name:   "Computer",
			Side:   chess.Black,
			Ready:  true,
		}
	} else {
		game.White = &Player{
			UserID: "computer",
			Name:   "Computer",
			Side:   chess.White,
			Ready:  true,
		}
		game.Black = &Player{
			UserID: userID,
			Name:   userName,
			Side:   chess.Black,
			Ready:  true,
		}
	}

	gm.games[game.ID] = game
	return game
}

func (gm *GameManager) CreateOpeningStudyGame(userID, userName string) *Game {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	game := &Game{
		ID:             gm.GenerateUniqueGameID(),
		chess:          chess.NewGame(),
		Status:         "active",
		CreatedAt:      time.Now(),
		Mode:           ModeOpeningStudy,
		AllowFreeMoves: true,
		SANHistory:     []string{},
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

	if game.Mode != ModeMultiplayer {
		return fmt.Errorf("only multiplayer games can be joined")
	}

	if game.Status != "waiting" {
		return fmt.Errorf("game already started")
	}

	if game.Black != nil {
		return fmt.Errorf("game already full")
	}

	game.Black = &Player{
		UserID: userID,
		Name:   userName,
		Side:   chess.Black,
		Ready:  true,
	}

	if game.White != nil && game.White.Ready && game.Black.Ready {
		game.Status = "active"
	}

	return nil
}

func (gm *GameManager) SurrenderGame(gameID string, surrenderingUserID string) (winner string, result string, fen string, err error) {
	gm.mutex.RLock()
	game, exists := gm.games[gameID]
	gm.mutex.RUnlock()

	if !exists {
		return "", "", "", fmt.Errorf("game not found")
	}

	game.mutex.Lock()
	defer game.mutex.Unlock()

	if game.Status == "finished" {
		return game.Winner, game.Result, game.chess.FEN(), nil
	}

	var surrenderingColor chess.Color
	var winnerName string

	switch {
	case game.White != nil && game.White.UserID == surrenderingUserID:
		surrenderingColor = chess.White
		if game.Black != nil {
			winnerName = game.Black.Name
		} else {
			winnerName = "Black"
		}
	case game.Black != nil && game.Black.UserID == surrenderingUserID:
		surrenderingColor = chess.Black
		if game.White != nil {
			winnerName = game.White.Name
		} else {
			winnerName = "White"
		}
	default:
		return "", "", "", fmt.Errorf("user is not part of this game")
	}

	game.Status = "finished"

	if surrenderingColor == chess.White {
		game.Winner = "black"
		game.Result = "White surrendered. Black wins."
	} else {
		game.Winner = "white"
		game.Result = "Black surrendered. White wins."
	}

	if winnerName != "" {
		if surrenderingColor == chess.White {
			game.Result = fmt.Sprintf("%s wins by resignation.", winnerName)
		} else {
			game.Result = fmt.Sprintf("%s wins by resignation.", winnerName)
		}
	}

	return game.Winner, game.Result, game.chess.FEN(), nil
}

func (gm *GameManager) ApplySANMove(gameID, moveStr string) (string, error) {
	gm.mutex.RLock()
	game, exists := gm.games[gameID]
	gm.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("game not found")
	}

	game.mutex.Lock()
	defer game.mutex.Unlock()

	if game.Status == "finished" {
		return "", fmt.Errorf("game is already finished")
	}

	move, err := chess.AlgebraicNotation{}.Decode(game.chess.Position(), moveStr)
	if err != nil {
		return "", err
	}

	if err := game.chess.Move(move); err != nil {
		return "", err
	}

	game.SANHistory = append(game.SANHistory, moveStr)
	return game.chess.FEN(), nil
}

func (gm *GameManager) UndoLastMove(gameID string) (string, error) {
	gm.mutex.RLock()
	game, exists := gm.games[gameID]
	gm.mutex.RUnlock()

	if !exists {
		return "", fmt.Errorf("game not found")
	}

	game.mutex.Lock()
	defer game.mutex.Unlock()

	moves := game.chess.Moves()
	if len(moves) == 0 {
		return game.chess.FEN(), nil
	}

	newGame := chess.NewGame()
	for i := 0; i < len(moves)-1; i++ {
		if err := newGame.Move(moves[i]); err != nil {
			return "", err
		}
	}

	game.chess = newGame
	if len(game.SANHistory) > 0 {
		game.SANHistory = game.SANHistory[:len(game.SANHistory)-1]
	}

	return game.chess.FEN(), nil
}

func (gm *GameManager) GetGame(gameID string) *Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()
	return gm.games[gameID]
}

func (gm *GameManager) CleanupOldGames() {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	for id, game := range gm.games {
		if game.Mode == ModeMultiplayer && game.Black == nil && time.Since(game.CreatedAt) > time.Hour {
			delete(gm.games, id)
			continue
		}

		if (game.Mode == ModeVsComputer || game.Mode == ModeOpeningStudy) && time.Since(game.CreatedAt) > 4*time.Hour {
			delete(gm.games, id)
		}
	}
}

func (gm *GameManager) GetActiveGames() []*Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	var activeGames []*Game
	for _, game := range gm.games {
		if game.Status == "active" {
			activeGames = append(activeGames, game)
		}
	}
	return activeGames
}

func (gm *GameManager) GetWaitingGames() []*Game {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	var waitingGames []*Game
	for _, game := range gm.games {
		if game.Status == "waiting" {
			waitingGames = append(waitingGames, game)
		}
	}
	return waitingGames
}

func (gm *GameManager) GenerateUniqueGameID() string {
	for {
		b := make([]byte, 4)
		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}
		id := hex.EncodeToString(b)

		if _, exists := gm.games[id]; !exists {
			return id
		}
	}
}
