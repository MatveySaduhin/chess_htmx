package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) AdminPage(c *gin.Context) {
	data := s.newTemplateData(c)
	data["Title"] = "Admin Panel"

	users, err := s.authService.GetAllUsers()
	if err != nil {
		log.Printf("Error fetching users for admin panel: %v", err)
		c.String(http.StatusInternalServerError, "Could not fetch users")
		return
	}
	for i := range users {
		if !users[i].CreatedAt.IsZero() {
			users[i].FormattedDate = users[i].CreatedAt.Format("Jan 02, 2006")
		} else {
			users[i].FormattedDate = "Unknown"
		}
	}
	data["AllUsers"] = users

	activeGames := s.gameManager.GetActiveGames()
	waitingGames := s.gameManager.GetWaitingGames()

	for _, game := range activeGames {
		game.FormattedCreatedAt = game.CreatedAt.Format("Jan 02, 2006 15:04")
	}
	for _, game := range waitingGames {
		game.FormattedCreatedAt = game.CreatedAt.Format("Jan 02, 2006 15:04")
	}

	data["ActiveGames"] = activeGames
	data["WaitingGames"] = waitingGames
	data["TotalGames"] = len(activeGames) + len(waitingGames)

	c.HTML(http.StatusOK, "admin.html", data)
}

func (s *Server) RenameUser(c *gin.Context) {
    var request struct {
        UserID  string `json:"user_id" binding:"required"`
        NewName string `json:"new_name" binding:"required,min=2,max=50"`
    }

    if err := c.ShouldBindJSON(&request); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request: " + err.Error()})
        return
    }

    err := s.authService.RenameUser(request.UserID, request.NewName)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "message": "User renamed successfully",
        "new_name": request.NewName,
    })
}
