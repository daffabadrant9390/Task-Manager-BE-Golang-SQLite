package models

import (
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id" gorm:"primaryKey"`
	Username string `json:"username" gorm:"unique;not null"`
	Password string `json:"-" gorm:"not null"`
	gorm.Model
}

// TableName specifies the table name for User Model
func (User) TableName() string {
	return "users"
}
