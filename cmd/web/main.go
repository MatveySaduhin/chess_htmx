package main

import (
	"chess_htmx/internal/api"
	"chess_htmx/internal/game"
	"chess_htmx/internal/wsmanager"
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
// NOTE: Creating MongoDB client (I don't like it like that - probably should implement init function)
    ctx := context.Background()
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
	log.Fatal("Failed to connect to MongoDB: ", err)
    }
    defer func() {
	if err := client.Disconnect(ctx); err != nil {
	    log.Fatal("Failed to disconnect from MongoDB", err)
	}
    }()

    authService, err:= api.NewAuthService(client, "my-mongoDB")
    if err != nil {
	log.Fatal("Errror initializing authentication service: ", err)
    }

    sessionService := api.NewSessionService(client, "chess_app", "your-secret-key-here")

// NOTE: Temporary user seed
    authService.NewUser("Alice", "alice@ex.com", "1234567")
    authService.NewUser("Bob", "bob@ex.com", "1234567")

    gameManager := game.NewGameManager()

    handlers := api.NewHandlers(gameManager, authService, sessionService)

    wsHub := wsmanager.NewHub(wsmanager.Config{})
    go wsHub.Run()

    r := gin.Default()

    r.Static("/static", "./static")
    r.LoadHTMLGlob("templates/*")

    r.GET("/", handlers.Home)
    r.GET("/auth", handlers.AuthPage)
    r.GET("/registration", handlers.RegPage)
    r.POST("/api/login", handlers.Login)

    protected := r.Group("/")
    protected.Use(handlers.AuthMiddleware())
    {
	protected.POST("/api/create-game", handlers.CreateGame)
	protected.POST("/api/registration", handlers.Register)
	protected.POST("/api/logout", handlers.Logout)
	protected.POST("/api/join-game", handlers.JoinGame)
    	protected.GET("/game/:id", handlers.GamePage)
    	protected.GET("/ws", func(c *gin.Context) {
    	    wsHub.ServeWebSocket(c)
    	})
    }

    if err := r.Run(":8080"); err != nil {
        log.Fatal("Error starting server: ", err)
    }
}
