package wsmanager

import (
    "log"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

type GameHub struct {
    games      map[string]*GameRoom
    register   chan *Client
    unregister chan *Client
    upgrader   *websocket.Upgrader
}

type GameRoom struct {
    clients   map[*Client]bool
    broadcast chan []byte
    gameID    string
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

func NewGameHub(config Config) *GameHub {
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
    // Get or create game room
    room, exists := h.games[client.gameID]
    if !exists {
        room = &GameRoom{
            clients: make(map[*Client]bool),
            broadcast: make(chan []byte),
            gameID: client.gameID,
        }
        h.games[client.gameID] = room
        go room.run()
    }
    room.clients[client] = true
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
    defer func() {
        c.conn.Close()
    }()

    for message := range c.send {
        err := c.conn.WriteMessage(websocket.TextMessage, message)
        if err != nil {
            log.Printf("Write error: %v", err)
            return
        }
    }

    // Channel was closed - send close message
    c.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
