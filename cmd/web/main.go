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
    client := initMongoDB()
    defer func() {
	if err := client.Disconnect(context.Background()); err != nil {
		log.Printf("Warning: Failed to disconnect from MongoDB: %v", err)
	}
    }()

    authService := initAuthService(client)

    sessionService := api.NewSessionService(client, "chess_app", "your-secret-key-here")

    gameManager := game.NewGameManager()

    websocketHub := initWebSocketHub()
    go websocketHub.Run()

    handlers := api.NewServer(gameManager, authService, sessionService, websocketHub)

    router := setupRouter(handlers)
    if err := router.Run(":8080"); err != nil {
        log.Fatal("Error starting server: ", err)
    }
}

func setupRouter(handlers *api.Server) *gin.Engine {
	r := gin.Default()

	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*")

	public := r.Group("/")
	{
		public.GET("/", handlers.Home)
		public.GET("/auth", handlers.AuthPage)
		public.GET("/registration", handlers.RegPage)
		public.POST("/api/login", handlers.Login)
	}

	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.POST("/api/create-game", handlers.CreateGame)
		protected.POST("/api/registration", handlers.Register)
		protected.POST("/api/logout", handlers.Logout)
		protected.POST("/api/join-game", handlers.JoinGame)
		protected.GET("/game/:id", handlers.GamePage)
		protected.GET("/ws/game/:id", handlers.GameWebsocket)
	}

	return r
}

func initMongoDB() *mongo.Client {
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
    if err != nil {
	log.Fatal("Failed to connect to MongoDB: ", err)
    }
    if err := client.Ping(context.Background(), nil); err != nil {
	log.Fatal("Failed to ping MongoDB: ", err)
    }
    return client
}

func initAuthService(client *mongo.Client) *api.AuthService {
    authService, err := api.NewAuthService(client, "chess_app")
    if err != nil {
    	log.Fatal("Error initializing authentication service: ", err)
    }
    // Seed users - log errors but don't fail
    if _, err := authService.NewUser("Alice", "alice@ex.com", "1234567"); err != nil {
    	log.Printf("Note: Failed to create Alice: %v", err)
    }
    if _, err := authService.NewUser("Bob", "bob@ex.com", "1234567"); err != nil {
    	log.Printf("Note: Failed to create Bob: %v", err)
    }
    return authService
}

func initWebSocketHub() *wsmanager.GameHub {
	hub := wsmanager.NewGameHub(wsmanager.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	})
	go hub.Run()
	return hub
}

