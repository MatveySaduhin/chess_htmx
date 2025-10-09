package wsmanager

import (
    "log"
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

type Config struct {
    CheckOrigin func(r *http.Request) bool
    ReadBufferSize int
    WriteBufferSize int
}

type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte
}

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    upgrader   *websocket.Upgrader
}

func NewHub(config Config) *Hub {
    upgrader := &websocket.Upgrader{
        CheckOrigin:     config.CheckOrigin,
        ReadBufferSize:  config.ReadBufferSize,
        WriteBufferSize: config.WriteBufferSize,

    }

    if upgrader.CheckOrigin == nil {
        // allow only same origin
        upgrader.CheckOrigin = func(r *http.Request) bool {
            return r.Header.Get("Origin") == "trusted-domain.com"
        }
    }

    return &Hub{
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
        upgrader:   upgrader,
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}

func (hub *Hub) ServeWebSocket (c *gin.Context) {
    conn, err := hub.upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Println("WebSocket upgrade failed:", err)
        return
    }

    client := &Client{
        hub:  hub,
        conn: conn,
        send: make(chan []byte, 256),
    }

    client.hub.register <- client
    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c  // Unregister when done
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
            // Process the message (in your case, chess moves)
            c.hub.broadcast <- message
        }
    }
}

func (c *Client) writePump() {
    defer func() {
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                // Hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            err := c.conn.WriteMessage(websocket.TextMessage, message)
            if err != nil {
                log.Printf("Write error: %v", err)
                return
            }
        }
    }
}
