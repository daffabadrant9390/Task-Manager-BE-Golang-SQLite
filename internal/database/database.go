package database

import (
	"log"
	"task-management-api/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDB initializes the database connection and runs migrations
func InitDB() {
	var err error

	// Open SQLite database file (will be created if it doesn't exist initially)
	// Using glebarez/sqlite which is a pure Go implementation (no CGO required)
	DB, err = gorm.Open(sqlite.Open("tasks-management.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	// Auto-migrate the schema (it will create tables if they don't exist)
	err = DB.AutoMigrate(
		&models.User{},
		&models.Task{},
	)

	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database connected and migrated successfully!!!")
}

// GetDB returns the database connection
func GetDB() *gorm.DB {
	return DB
}