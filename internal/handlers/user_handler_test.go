package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"task-management-api/internal/auth"
	"task-management-api/internal/database"
	"task-management-api/internal/middleware"
	"task-management-api/internal/models"
	"task-management-api/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetAllUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := testutil.NewInMemoryDB()
	require.NoError(t, err)
	database.DB = db

	// Seed some users
	_ = db.Create(&models.User{ID: "u-1", Username: "alice", Password: "x"}).Error
	_ = db.Create(&models.User{ID: "u-2", Username: "bob", Password: "x"}).Error

	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware())
	r.GET("/api/users", GetAllUsers)

	token, _ := auth.GenerateToken("u-1", "alice")
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}


