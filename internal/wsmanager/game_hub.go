package wsmanager

import (
	"chess_htmx/internal/game"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/notnil/chess"
)

type GameHub struct {
	games       map[string]*GameRoom
	register    chan *Client
	unregister  chan *Client
	upgrader    *websocket.Upgrader
	gameManager *game.GameManager
}

type GameRoom struct {
	clients        map[*Client]bool
	broadcast      chan []byte
	gameID         string
	game           *chess.Game
	whiteConnected bool
	blackConnected bool
}

type Client struct {
	hub    *GameHub
	conn   *websocket.Conn
	send   chan []byte
	gameID string
	userID string
}

type Config struct {
	CheckOrigin     func(r *http.Request) bool
	ReadBufferSize  int
	WriteBufferSize int
}

type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type MoveMessage struct {
	Move string `json:"move"`
	//  Promotion string `json:"promotion,omitempty"`
	FEN string `json:"fen"`
}

type GameStateMessage struct {
	FEN string `json:"fen"`
}

type ErrorMessage struct {
	Error string `json:"error"`
}

func NewGameHub(config Config, gm *game.GameManager) *GameHub {
	upgrader := &websocket.Upgrader{
		CheckOrigin:     config.CheckOrigin,
		ReadBufferSize:  config.ReadBufferSize,
		WriteBufferSize: config.WriteBufferSize,
	}
	// NOTE: Allow all for development
	if upgrader.CheckOrigin == nil {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true // Allow all for development
		}
	}

	return &GameHub{
		games:       make(map[string]*GameRoom),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		upgrader:    upgrader,
		gameManager: gm,
	}
}

func (h *GameHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)
		case client := <-h.unregister:
			h.unregisterClient(client)
		}
	}
}

func (r *GameRoom) run() {
	for message := range r.broadcast {
		for client := range r.clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(r.clients, client)
			}
		}
	}

	// Clean up when broadcast channel is closed
	for client := range r.clients {
		close(client.send)
		delete(r.clients, client)
	}
}

func (h *GameHub) registerClient(client *Client) {
	log.Printf("registerClient: gameID=%s, userID=%s", client.gameID, client.userID)

	room, exists := h.games[client.gameID]
	if !exists {
		// Always create the room - the game might be created later
		room = &GameRoom{
			clients:   make(map[*Client]bool),
			broadcast: make(chan []byte),
			gameID:    client.gameID,
			game:      chess.NewGame(), // Start with fresh chess game
		}
		h.games[client.gameID] = room
		go room.run()
		log.Printf("Room created for game: %s", client.gameID)
	}

	room.clients[client] = true

	color := h.gameManager.GetPlayerColor(client.userID, client.gameID)
	if color == chess.White {
		room.whiteConnected = true
	} else {
		room.blackConnected = true
	}

	log.Printf("Player %s (%s) connected. White: %t, Black: %t",
		client.userID, color, room.whiteConnected, room.blackConnected)

	h.sendGameState(client)
	log.Printf("Client added to room. Total clients: %d", len(room.clients))

	if room.whiteConnected && room.blackConnected {
		h.broadcastGameStart(room)
	} else {
		h.broadcastWaiting(room)
	}
}

func (h *GameHub) broadcastPlayerDisconnected(room *GameRoom, color chess.Color) {
	message := WSMessage{
		Type: "player_disconnected",
		Payload: map[string]interface{}{
			"message": fmt.Sprintf("%s player disconnected", color.Name()),
			"color":   strings.ToLower(color.Name()),
			"fen":     room.game.FEN(),
		},
	}

	msgBytes, _ := json.Marshal(message)
	room.broadcast <- msgBytes

	log.Printf("Broadcasted %s player disconnect in game %s", color.Name(), room.gameID)
}

func (h *GameHub) unregisterClient(client *Client) {
	if room, exists := h.games[client.gameID]; exists {
		if _, clientExists := room.clients[client]; clientExists {
			// Update connection status
			color := h.gameManager.GetPlayerColor(client.userID, client.gameID)
			if color == chess.White {
				room.whiteConnected = false
			} else {
				room.blackConnected = false
			}

			delete(room.clients, client)
			close(client.send)

			// Notify other player about disconnection
			h.broadcastPlayerDisconnected(room, color)
			if len(room.clients) == 0 {
				delete(h.games, client.gameID)
			}
		}
	}
}

func (hub *GameHub) ServeGameWebSocket(c *gin.Context, gameID, userID string) {
	conn, err := hub.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}

	client := &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		gameID: gameID,
		userID: userID,
	}

	hub.register <- client
	go client.writePump()
	go client.readPump()
}

func (h *GameHub) handleGameMessage(client *Client, message []byte) {
	log.Printf("handleGameMessage called for client %s", client.userID)
	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		log.Printf("Invalid message format: %v", err)
		return
	}

	switch wsMsg.Type {
	case "move":
		h.handleMoveMessage(client, wsMsg.Payload)
	case "chat":
		h.handleChatMessage(client, wsMsg.Payload)
	default:
		log.Printf("Unknown message type: %s", wsMsg.Type)
	}
}

func (h *GameHub) handleMoveMessage(client *Client, payload any) {
	moveStr, ok := payload.(string)
	if !ok {
		return
	}

	room, exists := h.games[client.gameID]
	if !exists || room.game == nil {
		return
	}

	// Parse using the standard notation
	move, err := chess.AlgebraicNotation{}.Decode(room.game.Position(), moveStr)
	if err != nil {
		h.sendError(client, "Invalid move: "+err.Error())
		return
	}

	// Apply the move
	if err := room.game.Move(move); err != nil {
		h.sendError(client, "Invalid move: "+err.Error())
		return
	}

	// Broadcast the SAN move to all clients
	response := WSMessage{
		Type: "move",
		Payload: MoveMessage{
			Move: moveStr, // Send back the same SAN
			FEN:  room.game.FEN(),
		},
	}

	msgBytes, _ := json.Marshal(response)
	room.broadcast <- msgBytes
}

func (h *GameHub) sendError(client *Client, errorMsg string) {
	errorResponse := WSMessage{
		Type: "error",
		Payload: ErrorMessage{
			Error: errorMsg,
		},
	}
	msgBytes, _ := json.Marshal(errorResponse)
	client.send <- msgBytes
}

func (h *GameHub) handleChatMessage(client *Client, payload interface{}) {
	chatData, ok := payload.(map[string]interface{})
	if !ok {
		log.Printf("Invalid chat payload")
		return
	}

	message, ok := chatData["message"].(string)
	if !ok || message == "" {
		log.Printf("Invalid chat message")
		return
	}

	// Broadcast chat message to all clients in the room
	if room, exists := h.games[client.gameID]; exists {
		response := WSMessage{
			Type: "chat",
			Payload: map[string]interface{}{
				"message":   message,
				"userID":    client.userID,
				"userName":  h.getUserName(client.userID), // You'll need to implement this
				"timestamp": time.Now().Unix(),
			},
		}

		msgBytes, _ := json.Marshal(response)
		room.broadcast <- msgBytes
	}
}

// Helper method to get user name (you'll need to implement this based on your auth service)
func (h *GameHub) getUserName(userID string) string {
	// This depends on how you access your user data
	// You might need to pass authService to GameHub or use a different approach
	return "Player" // Placeholder
}

func (h *GameHub) sendGameState(client *Client) {
	room, exists := h.games[client.gameID]
	if !exists || room.game == nil {
		return
	}

	// Just send the FEN - it's the complete game state
	gameState := WSMessage{
		Type: "game_state",
		Payload: map[string]string{
			"fen": room.game.FEN(),
		},
	}

	msgBytes, _ := json.Marshal(gameState)
	client.send <- msgBytes
}

func (h *GameHub) broadcastGameStart(room *GameRoom) {
	message := WSMessage{
		Type: "game_start",
		Payload: map[string]interface{}{
			"message": "Both players connected! Game starting...",
			"fen":     room.game.FEN(),
		},
	}

	msgBytes, _ := json.Marshal(message)
	room.broadcast <- msgBytes
}

func (h *GameHub) broadcastWaiting(room *GameRoom) {
	message := WSMessage{
		Type: "game_waiting",
		Payload: map[string]interface{}{
			"message":        "Waiting for opponent to connect...",
			"whiteConnected": room.whiteConnected,
			"blackConnected": room.blackConnected,
		},
	}

	msgBytes, _ := json.Marshal(message)
	room.broadcast <- msgBytes
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			log.Printf("Recieved raw data from the client %s: %s", c.userID, string(message))
			c.hub.handleGameMessage(c, message)
		}
	}
}

func (c *Client) writePump() {
	log.Printf("📝 Write pump started for client %s", c.userID)
	defer func() {
		log.Printf("📝 Write pump stopped for client %s", c.userID)
		c.conn.Close()
	}()

	for message := range c.send {
		err := c.conn.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			log.Printf("❌ Write error for client %s: %v", c.userID, err)
			return
		}
		log.Printf("✅ Client %s: successfully wrote message to client", c.userID)
	}

	log.Printf("🔌 Client %s send channel closed", c.userID)
	c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
