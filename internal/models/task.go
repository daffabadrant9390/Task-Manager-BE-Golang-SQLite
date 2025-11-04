package models

import (
	"gorm.io/gorm"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusTodo TaskStatus = "todo"
	StatusInProgress TaskStatus = "inProgress"
	StatusDone	TaskStatus = "done"
)

// Task Priority represents the priority of a task
type TaskPriority string

const (
	PriorityHigh TaskPriority = "high"
	PriorityMedium TaskPriority = "medium"
	PriorityLow TaskPriority = "low"
)

// TaskType represents the type of a task (story, defect, subtask)
type TaskType string

const (
	TypeStory TaskType = "story"
	TypeDefect TaskType = "defect"
	TypeSubtask TaskType = "subtask"
)

// Assignee represents a task assignee
type Assignee struct {
	ID string `json:"id"`
	Name string `json:"name"`
}

// Task represents a task in the system
type Task struct {
	ID          string      `json:"id" gorm:"primaryKey"`
	Title       string      `json:"title" gorm:"not null"`
	Description string      `json:"description"`
	Status      TaskStatus  `json:"status" gorm:"not null;default:'todo'"`
	ProjectID   string      `json:"projectId" gorm:"column:project_id"`
	AssigneeID  string      `json:"-" gorm:"column:assignee_id"`
	Assignee    Assignee    `json:"assignee" gorm:"-"`
	StartDate   string      `json:"startDate" gorm:"column:start_date"`
	EndDate     string      `json:"endDate" gorm:"column:end_date"`
	Effort      int         `json:"effort" gorm:"default:1"`
	Priority    TaskPriority `json:"priority" gorm:"default:'medium'"`
	TaskType    TaskType    `json:"taskType" gorm:"column:task_type;default:'story'"`
	UserID      string      `json:"-" gorm:"column:user_id;index"`
	gorm.Model
}

// TableName specifies the table name for Task Model
func(Task) TableName() string {
	return "tasks"
}