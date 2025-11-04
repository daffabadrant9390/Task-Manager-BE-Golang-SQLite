package handlers

import (
	"net/http"
	"task-management-api/internal/auth"
	"task-management-api/internal/database"
	"task-management-api/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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

// Login handles the login endpoint with unique username and password verification
// Password provided by FE is a SHA-256 hash of the original password.
// We store and verify using bcrypt(hashFromFE).
// POST /api/login
func Login(c *gin.Context) {
    var req LoginRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request. Username and password are required.",
        })
        return
    }

    db := database.GetDB()

    // Find user by username
    var user models.User
    result := db.Where("username = ?", req.Username).First(&user)

    if result.Error == nil {
        // Username exists → verify password (bcrypt compare)
        if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
            return
        }

        token, err := auth.GenerateToken(user.ID, user.Username)
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
            return
        }

        c.JSON(http.StatusOK, LoginResponse{
            Token:    token,
            UserID:   user.ID,
            Username: user.Username,
            Message:  "Login successful",
        })
        return
    }

    // Username not found → create new user with bcrypt-hashed FE password
    hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
        return
    }

    newUser := models.User{
        ID:       "user-" + time.Now().Format("20060102150405.000000000"),
        Username: req.Username,
        Password: string(hashed),
    }

    if err := db.Create(&newUser).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
        return
    }

    token, err := auth.GenerateToken(newUser.ID, newUser.Username)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(http.StatusOK, LoginResponse{
        Token:    token,
        UserID:   newUser.ID,
        Username: newUser.Username,
        Message:  "Signup & login successful",
    })
}