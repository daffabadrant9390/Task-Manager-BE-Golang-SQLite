package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := SetupRoutes()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}


