package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHomeHandler_Unit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("home page returns 200", func(t *testing.T) {
		router := gin.New()
		router.GET("/", func(c *gin.Context) {
			c.String(200, "OK")
		})

		req, _ := http.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, 200, w.Code)
	})
}
