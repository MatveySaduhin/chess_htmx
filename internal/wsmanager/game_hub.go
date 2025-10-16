package wsmanager

import (
	"chess_htmx/internal/game"
	"encoding/json"
	"log"
	"net/http"
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
    clients   map[*Client]bool
    broadcast chan []byte
    gameID    string
    game      *chess.Game
}

type Client struct {
    hub  *GameHub
    conn *websocket.Conn
    send chan []byte
    gameID string
    userID string
}

type Config struct {
    CheckOrigin func(r *http.Request) bool
    ReadBufferSize int
    WriteBufferSize int
}

type WSMessage struct {
    Type    string `json:"type"`
    Payload any    `json:"payload"`
}

type MoveMessage struct {
    From      string `json:"from"`
    To        string `json:"to"`
    Promotion string `json:"promotion,omitempty"`
    FEN       string `json:"fen"`
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
        games:      make(map[string]*GameRoom),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        upgrader:   upgrader,
        gameManager:     gm,
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
        log.Printf("Creating new room for game: %s", client.gameID)
        
        // Always create the room - the game might be created later
        room = &GameRoom{
            clients: make(map[*Client]bool),
            broadcast: make(chan []byte),
            gameID: client.gameID,
            game: chess.NewGame(), // Start with fresh chess game
        }
        h.games[client.gameID] = room
        go room.run()
        
        log.Printf("Room created for game: %s", client.gameID)
    }
    
    room.clients[client] = true
    log.Printf("Client added to room. Total clients: %d", len(room.clients))
}

func (h *GameHub) unregisterClient(client *Client) {
    if room, exists := h.games[client.gameID]; exists {
        if _, clientExists := room.clients[client]; clientExists {
            delete(room.clients, client)
            close(client.send)

            // Clean up empty rooms
            if len(room.clients) == 0 {
                delete(h.games, client.gameID)
            }
        }
    }
}

func (hub *GameHub) ServeGameWebSocket (c *gin.Context, gameID, userID string) {
    conn, err := hub.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Println("WebSocket upgrade failed:", err)
        return
    }

    client := &Client{
        hub:  hub,
        conn: conn,
        send: make(chan []byte, 256),
        gameID: gameID,
        userID: userID,
    }

    hub.register <- client
    go client.writePump()
    go client.readPump()
}

func (h *GameHub) handleGameMessage(client *Client, message []byte) {
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

func (h *GameHub) handleMoveMessage(client *Client, payload interface{}) {
    log.Printf("handleMoveMessage: client %s in game %s", client.userID,client.gameID)

    moveData, ok := payload.(map[string]interface{})
    if !ok {
        log.Printf("Invalid move payload")
        return
    }

    from, ok1 := moveData["from"].(string)
    to, ok2 := moveData["to"].(string)
    if !ok1 || !ok2 {
        log.Printf("Invalid move coordinates")
        return
    }

    room, exists := h.games[client.gameID]
    if !exists || room.game == nil {
        log.Printf("Game not found or not initialized")
        return
    }

    log.Printf("Processing move: %s to %s", from, to)
    // Create move string (e.g., "e2e4", "e7e8q")
    moveStr := from + to
    if promotion, ok := moveData["promotion"].(string); ok && promotion != "" {
        moveStr += promotion
    }

    // Parse the move
    move, err := chess.LongAlgebraicNotation{}.Decode(room.game.Position(), moveStr)
    if err != nil {
        h.sendError(client, "Invalid move format: "+err.Error())
        return
    }

    // Try to apply the move - this validates it automatically
    if err := room.game.Move(move); err != nil {
        h.sendError(client, "Invalid move: "+err.Error())
        return
    }

    // Move is valid - broadcast to all clients
    response := WSMessage{
        Type: "move",
        Payload: MoveMessage{
            From: from,
            To: to,
            Promotion: move.Promo().String(),
            FEN: room.game.FEN(), // Send updated position
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
                "message": message,
                "userID":  client.userID,
                "userName": h.getUserName(client.userID), // You'll need to implement this
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
            // Broadcast to all clients in the same game room
            if room, exists := c.hub.games[c.gameID]; exists {
                room.broadcast <- message
            }
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
        log.Printf("📨 Client %s writing message to WebSocket", c.userID)
        err := c.conn.WriteMessage(websocket.TextMessage, message)
        if err != nil {
            log.Printf("❌ Write error for client %s: %v", c.userID, err)
            return
        }
        log.Printf("✅ Client %s successfully wrote message", c.userID)
    }

    log.Printf("🔌 Client %s send channel closed", c.userID)
    c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}

