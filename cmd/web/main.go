package main

import (
	"chess_htmx/internal/game"
	"chess_htmx/internal/wsmanager"
        "chess_htmx/internal/api"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
    gameManager := game.NewGameManager()
    handlers := api.NewHandlers(gameManager)
    wsHub := wsmanager.NewHub(wsmanager.Config{})

    go wsHub.Run()

    r := gin.Default()

    r.Static("/static", "./static")
    r.LoadHTMLGlob("templates/*")

    r.GET("/", handlers.Home)

    r.POST("/api/games", handlers.CreateGame)
    r.GET("/game/:id", func(c *gin.Context) {})

    r.GET("/auth", handlers.AuthPage)
    r.POST("/api/login", handlers.Login)

    r.GET("/ws", func(c *gin.Context) {
        wsHub.ServeWebSocket(c)
    })

    if err := r.Run(":8080"); err != nil {
        log.Fatal("Error starting server: ", err)
    }
}
