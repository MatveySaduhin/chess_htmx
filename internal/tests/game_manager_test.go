package tests

import (
	"testing"
    "chess_htmx/internal/game"
)

func TestGenerateGameID(t *testing.T) {
	gm := game.NewGameManager()
	id := gm.GenerateUniqueGameID()

	if len(id) != 6 {
		t.Errorf("Expected game ID length 6, got %d", len(id))
	}
}

func TestGameManager_CreateGame(t *testing.T) {
	gm := game.NewGameManager()
	game := gm.CreateGame("user123", "Test User")

	if game == nil {
		t.Error("Expected game to be created, got nil")
	}

	if game.White == nil {
		t.Error("Expected white player to be set")
	}

	if game.White.UserID != "user123" {
		t.Errorf("Expected user ID 'user123', got '%s'", game.White.UserID)
	}
}

func TestGameManager_FindAvailableGame(t *testing.T) {
	gm := game.NewGameManager()

	// No games available initially
	available := gm.FindAvailableGame()
	if available != nil {
		t.Error("Expected no available games initially")
	}

	// Create a game and it should be available
	gm.CreateGame("user1", "User 1")
	available = gm.FindAvailableGame()
	if available == nil {
		t.Error("Expected to find available game after creation")
	}
}
