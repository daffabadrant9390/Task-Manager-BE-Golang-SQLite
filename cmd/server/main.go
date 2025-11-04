package main

import (
	"log"
	"task-management-api/internal/database"
	"task-management-api/internal/routes"
)

func main() {
	// Init database
	database.InitDB()

	// Setup the routes (public and protected routes)
	ginRoutes := routes.SetupRoutes()

	// Start server
	port := ":8008" // This is customizable based on the environment
	log.Printf("Server starting on port %s", port)
	log.Println("API endpoints:")
	log.Println("  POST   /api/login")
	log.Println("  GET    /api/tasks")
	log.Println("  GET    /api/tasks/:id")
	log.Println("  POST   /api/tasks")
	log.Println("  PUT    /api/tasks/:id")
	log.Println("  PATCH  /api/tasks/:id/status")
	log.Println("  DELETE /api/tasks/:id")
	log.Println("  GET    /health")

	if err := ginRoutes.Run(port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}