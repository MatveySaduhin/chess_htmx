package main

import (
	"chess_htmx/internal/api"
	"chess_htmx/internal/game"
	"chess_htmx/internal/wsmanager"
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	//  Get env variables
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/chess"
	}
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "fallback-secret-change-in-production"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	client := initMongoDB(mongoURI)
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("Warning: Failed to disconnect from MongoDB: %v", err)
		}
	}()

	authService := initAuthService(client)

	sessionService := api.NewSessionService(client, "chess_app", "your-secret-key-here")

	gameManager := game.NewGameManager()
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				gameManager.CleanupOldGames()
				log.Println("Cleaned up old games")
			}
		}
	}()

	websocketHub := initWebSocketHub(gameManager)
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
	//	 NOTE: func below is here only for dev
	//       TODO: proper cache handling for release
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/") {
			// No cache for development
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}
		c.Next()
	})

	r.LoadHTMLGlob("templates/*")

	public := r.Group("/")
	{
		public.GET("/", handlers.Home)
		public.GET("/auth", handlers.AuthPage)
		public.GET("/registration", handlers.RegPage)
		public.POST("/api/login", handlers.Login)
		public.POST("/api/registration", handlers.Register)
	}

	protected := r.Group("/")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.POST("/api/create-game", handlers.CreateGame)
		protected.POST("/api/logout", handlers.Logout)
		protected.POST("/api/quick-play", handlers.QuickPlay)
	
		protected.POST("/api/single-player/create", handlers.CreateSinglePlayerGame)
		protected.POST("/api/single-player/move", handlers.SinglePlayerMove)
		protected.POST("/api/single-player/undo", handlers.UndoSinglePlayerMove)
		protected.GET("/api/opening", handlers.GetOpeningInfo)
	
		protected.POST("/api/engine/move", handlers.EngineMove)
	
		protected.POST("/api/join-game", handlers.JoinGame)
		protected.POST("/api/game/surrender", handlers.SurrenderGame)
		protected.GET("/game/:id", handlers.GamePage)
		protected.GET("/ws/game/:id", handlers.GameWebsocket)
	
		admin := protected.Group("/admin")
		admin.Use(handlers.AdminMiddleware())
		{
			admin.GET("/", handlers.AdminPage)
			admin.POST("/rename-user", handlers.RenameUser)
		}
	}
	return r
}

func initMongoDB(URI string) *mongo.Client {
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(URI))
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
	if _, err := authService.NewUser("Alice", "alice@ex.com", "1234567", "user"); err != nil {
		log.Printf("Note: Failed to create Alice: %v", err)
	}
	if _, err := authService.NewUser("Bob", "bob@ex.com", "1234567", "user"); err != nil {
		log.Printf("Note: Failed to create Bob: %v", err)
	}
	if _, err := authService.NewUser("Kim", "kim@ex.com", "1234567", "user"); err != nil {
		log.Printf("Note: Failed to create Bob: %v", err)
	}
	if _, err := authService.NewUser("Harrier", "harrier@ex.com", "1234567", "user"); err != nil {
		log.Printf("Note: Failed to create Bob: %v", err)
	}
	return authService
}

func initWebSocketHub(gm *game.GameManager) *wsmanager.GameHub {
	hub := wsmanager.NewGameHub(wsmanager.Config{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}, gm)
	go hub.Run()
	return hub
}
