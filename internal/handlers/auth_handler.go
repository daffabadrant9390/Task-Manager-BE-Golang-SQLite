package handlers

import (
	"net/http"
	"task-management-api/internal/auth"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents the login request payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the login response
type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Message  string `json:"message"`
}

// Login handles the login endpoint (dummy authentication)
// POST /api/login
func Login(c *gin.Context) {
	var req LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request. Username and password are required.",
		})
		return
	}

	// Dummy authentication - accept any username/password
	// In production, you would validate against a database
	userID := "user-1"
	username := req.Username

	// Generate JWT token
	token, err := auth.GenerateToken(userID, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		UserID:   userID,
		Username: username,
		Message:  "Login successful",
	})
}