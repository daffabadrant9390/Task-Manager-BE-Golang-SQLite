package handlers

import (
	"bytes"
	"encoding/json"
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

func TestCreateTask_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := testutil.NewInMemoryDB()
	require.NoError(t, err)
	database.DB = db

	// Seed a user to be the assignee
	assignee := models.User{ID: "u-2", Username: "bob", Password: "x"}
	require.NoError(t, db.Create(&assignee).Error)

	r := gin.New()
	r.Use(middleware.JWTAuthMiddleware())
	r.POST("/api/tasks", CreateTask)

	token, err := auth.GenerateToken("u-1", "alice")
	require.NoError(t, err)

	payload := map[string]any{
		"title":       "Test Task",
		"description": "Desc",
		"assignee":    map[string]string{"id": assignee.ID, "name": assignee.Username},
		"startDate":   "2025-01-01",
		"endDate":     "2025-01-03",
		"taskType":    "story",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Task
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	require.Equal(t, 2, created.Effort) // 2025-01-01 to 2025-01-03 => 2 days
	require.Equal(t, assignee.ID, created.Assignee.ID)
}
