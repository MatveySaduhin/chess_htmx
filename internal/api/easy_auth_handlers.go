package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type easyAuthFormRequest struct {
	Name     string `form:"name" json:"name"`
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required"`
}

func (s *Server) EasyAuthRegister(c *gin.Context) {
	var req easyAuthFormRequest
	if err := bindFormOrJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.easyAuthClient.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeEasyAuthProxyError(c, err)
		return
	}

	displayName := strings.TrimSpace(req.Name)
	if displayName == "" {
		displayName = result.Email
	}

	profile, err := s.profileService.FindOrCreateByAuthUserID(c.Request.Context(), result.UserID, displayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_create_profile"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": result.UserID,
		"email":   result.Email,
		"profile": profile,
	})
}

func (s *Server) EasyAuthLogin(c *gin.Context) {
	var req easyAuthFormRequest
	if err := bindFormOrJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.easyAuthClient.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeEasyAuthProxyError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (s *Server) EasyAuthGamePage(c *gin.Context) {
	gameID := c.Param("id")
	c.HTML(http.StatusOK, "game.html", gin.H{
		"GameID":          gameID,
		"Color":           "white",
		"CurrentUserName": "",
		"WhitePlayerName": "",
		"BlackPlayerName": "",
		"Mode":            "multiplayer",
		"IsSinglePlayer":  false,
		"IsOpeningStudy":  false,
		"IsVsComputer":    false,
		"InitialFEN":      "start",
		"IsEasyAuth":      true,
	})
}

func (s *Server) EasyAuthGameWebsocket(c *gin.Context) {
	gameID := c.Param("id")
	s.websocketHub.ServeEasyAuthGameWebSocket(c, gameID, s.authenticateEasyAuthWebSocket)
}

func (s *Server) authenticateEasyAuthWebSocket(ctx context.Context, token string, gameID string) (string, string, error) {
	claims, err := s.easyAuthJWTValidator.Validate(ctx, strings.TrimSpace(token))
	if err != nil {
		return "", "", err
	}

	if !s.gameManager.CanUserAccessGame(claims.Subject, gameID) {
		return "", "", fmt.Errorf("access denied")
	}

	profile, err := s.profileService.FindOrCreateByAuthUserID(ctx, claims.Subject, "")
	if err != nil {
		return "", "", err
	}

	return claims.Subject, profile.DisplayName, nil
}

func (s *Server) EasyAuthProfileMe(c *gin.Context) {
	authUserID := c.GetString("auth_user_id")
	profile, err := s.profileService.FindOrCreateByAuthUserID(c.Request.Context(), authUserID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_load_profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
}

func (s *Server) EasyAuthGameState(c *gin.Context) {
	profile, ok := s.easyAuthProfile(c)
	if !ok {
		return
	}

	gameID := c.Param("id")
	gameObj := s.gameManager.GetGame(gameID)
	if gameObj == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "game_not_found"})
		return
	}

	if !s.gameManager.CanUserAccessGame(profile.AuthUserID, gameID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access_denied"})
		return
	}

	color := strings.ToLower(s.gameManager.GetPlayerColor(profile.AuthUserID, gameID).Name())
	whitePlayerName := ""
	blackPlayerName := ""
	if gameObj.White != nil {
		whitePlayerName = gameObj.White.Name
	}
	if gameObj.Black != nil {
		blackPlayerName = gameObj.Black.Name
	}

	mode := string(gameObj.Mode)
	c.JSON(http.StatusOK, gin.H{
		"game_id":             gameID,
		"color":               color,
		"current_user_name":   profile.DisplayName,
		"white_player_name":   whitePlayerName,
		"black_player_name":   blackPlayerName,
		"mode":                mode,
		"is_single_player":    mode != "multiplayer",
		"is_opening_study":    mode == "opening_study",
		"is_vs_computer":      mode == "vs_computer",
		"initial_fen":         gameObj.FEN(),
		"status":              gameObj.Status,
		"websocket_supported": true,
	})
}

func (s *Server) EasyAuthCreateGame(c *gin.Context) {
	profile, ok := s.easyAuthProfile(c)
	if !ok {
		return
	}

	gameObj := s.gameManager.CreateGame(profile.AuthUserID, profile.DisplayName)
	c.JSON(http.StatusOK, gin.H{
		"game_id": gameObj.ID,
		"color":   "white",
		"status":  gameObj.Status,
		"message": "Created game - waiting for opponent",
	})
}

func (s *Server) EasyAuthQuickPlay(c *gin.Context) {
	profile, ok := s.easyAuthProfile(c)
	if !ok {
		return
	}

	availableGame := s.gameManager.FindAvailableGame()
	if availableGame != nil && !s.gameManager.CanUserAccessGame(profile.AuthUserID, availableGame.ID) {
		if err := s.gameManager.JoinGame(availableGame.ID, profile.AuthUserID, profile.DisplayName); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"game_id": availableGame.ID,
			"color":   "black",
			"status":  "active",
			"message": "Joined existing game",
		})
		return
	}

	gameObj := s.gameManager.CreateGame(profile.AuthUserID, profile.DisplayName)
	c.JSON(http.StatusOK, gin.H{
		"game_id": gameObj.ID,
		"color":   "white",
		"status":  gameObj.Status,
		"message": "Created game - waiting for opponent",
	})
}

func (s *Server) EasyAuthJoinGame(c *gin.Context) {
	profile, ok := s.easyAuthProfile(c)
	if !ok {
		return
	}

	var req struct {
		GameID string `form:"game_id" json:"game_id" binding:"required"`
	}
	if err := bindFormOrJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "game_id is required"})
		return
	}

	if err := s.gameManager.JoinGame(req.GameID, profile.AuthUserID, profile.DisplayName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"game_id": req.GameID,
		"color":   "black",
		"status":  "active",
		"message": "Joined game",
	})
}

func (s *Server) easyAuthProfile(c *gin.Context) (*Profile, bool) {
	authUserID := c.GetString("auth_user_id")
	profile, err := s.profileService.FindOrCreateByAuthUserID(c.Request.Context(), authUserID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_load_profile"})
		return nil, false
	}
	return profile, true
}

func bindFormOrJSON(c *gin.Context, output any) error {
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return c.ShouldBindJSON(output)
	}
	return c.ShouldBind(output)
}

func writeEasyAuthProxyError(c *gin.Context, err error) {
	var easyAuthErr EasyAuthError
	if errors.As(err, &easyAuthErr) {
		c.Data(easyAuthErr.StatusCode, "application/json", []byte(easyAuthErr.Body))
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"error": "easy_auth_unavailable"})
}
