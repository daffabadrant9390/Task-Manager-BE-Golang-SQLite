package handlers

import (
	"net/http"
	"task-management-api/internal/database"
	"task-management-api/internal/models"

	"github.com/gin-gonic/gin"
)

type UserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// GetUsers returns all users (protected)
// GET /api/users
func GetAllUsers(c *gin.Context) {
	var users []models.User
	if err := database.GetDB().Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Map to safe response payload
	resp := make([]UserResponse, 0, len(users))
	for _, u := range users {
		resp = append(resp, UserResponse{
			ID:       u.ID,
			Username: u.Username,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": resp,
		"count": len(resp),
	})
}
