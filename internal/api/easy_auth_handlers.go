package api

import (
	"errors"
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

func (s *Server) EasyAuthProfileMe(c *gin.Context) {
	authUserID := c.GetString("auth_user_id")
	profile, err := s.profileService.FindOrCreateByAuthUserID(c.Request.Context(), authUserID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed_to_load_profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"profile": profile})
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
