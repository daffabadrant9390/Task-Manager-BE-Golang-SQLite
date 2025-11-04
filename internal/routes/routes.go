package routes

import (
	"task-management-api/internal/handlers"
	"task-management-api/internal/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	// Create a new GIN Router
	ginRouter := gin.Default()

	// CORS middleware (for frontend integration)
	ginRouter.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // This depends on the implementation of the frontend
			return
		}

		c.Next()
	})

	// Health check endpoint
	ginRouter.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
			"message": "Server Task Management API is running in Health Check Endpoint",
		})
	})

	// Public routes (no authentication required)
	api := ginRouter.Group("/api") 
	{
		// Login endpoint
		api.POST("/login", handlers.Login)
	}

	// Protected routes (authentication required)
	protectedRoutes := api.Group("")
	protectedRoutes.Use(middleware.JWTAuthMiddleware())
	{
		// Task endpoints
		protectedRoutes.GET("/tasks", handlers.GetTasks)
		protectedRoutes.GET("/tasks/:id", handlers.GetTaskByID)
		protectedRoutes.POST("/tasks", handlers.CreateTask)
		protectedRoutes.PUT("/tasks/:id", handlers.UpdateTask)
		protectedRoutes.PATCH("/tasks/:id/status", handlers.UpdateTaskStatus)
		protectedRoutes.DELETE("/tasks/:id", handlers.DeleteTask)
		// Users endpoint
		protectedRoutes.GET("/users", handlers.GetAllUsers)
	}

	return ginRouter
}