package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-management-api/internal/database"
	"task-management-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLogin_CreatesUserIfNotExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := testutil.NewInMemoryDB()
	require.NoError(t, err)
	database.DB = db

	r := gin.New()
	r.POST("/api/login", Login)

	body, _ := json.Marshal(map[string]string{
		"username": "newuser",
		"password": "sha256-from-fe",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct{ Token string }
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEmpty(t, resp.Token)
}
