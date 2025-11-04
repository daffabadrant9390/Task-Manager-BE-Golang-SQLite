package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"task-management-api/internal/database"
	"task-management-api/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateTaskRequest represents the request payload for creating a task
type CreateTaskRequest struct {
	Title       string                `json:"title" binding:"required"`
	Description string                `json:"description" binding:"required"`
	Status      models.TaskStatus     `json:"status"`
	ProjectID   string                `json:"projectId"`
	Assignee    models.Assignee       `json:"assignee" binding:"required"`
	StartDate   string                `json:"startDate" binding:"required"`
	EndDate     string                `json:"endDate" binding:"required"`
	Effort      int                   `json:"effort"`
	Priority    models.TaskPriority   `json:"priority"`
	TaskType    models.TaskType       `json:"taskType" binding:"required"`
}

// UpdateTaskRequest represents the request payload for updating a task
type UpdateTaskRequest struct {
	Title       *string                `json:"title"`
	Description *string                `json:"description"`
	Status      *models.TaskStatus     `json:"status"`
	ProjectID   *string                `json:"projectId"`
	Assignee    *models.Assignee       `json:"assignee"`
	StartDate   *string                `json:"startDate"`
	EndDate     *string                `json:"endDate"`
	Effort      *int                   `json:"effort"`
	Priority    *models.TaskPriority   `json:"priority"`
	TaskType    *models.TaskType       `json:"taskType"`
}

/**
	GetTasks handles GET /api/tasks
	Returns all tasks owned by the authenticated user
*/
func GetTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	var tasks []models.Task
	result := database.GetDB().Where("user_id = ?", userID).Find(&tasks)
	
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch tasks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"count": len(tasks),
	})
}

/**
	CreateTask handles POST /api/tasks
	Creates a new task for the authenticated user
*/
func CreateTask(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Set default values if not provided
	status := req.Status
	if status == "" {
		status = models.StatusTodo
	}

	priority := req.Priority
	if priority == "" {
		priority = models.PriorityMedium
	}

	effort := req.Effort
	if effort == 0 {
		effort = 1
	}

	projectID := req.ProjectID
	if projectID == "" {
		projectID = "AC-2015" // Default project ID
	}

	// Generate task ID (simple format: task-{timestamp})
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())

	// Create task
	task := models.Task{
		ID:          taskID,
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		ProjectID:   projectID,
		AssigneeID:  req.Assignee.ID,
		Assignee:    req.Assignee,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Effort:      effort,
		Priority:    priority,
		TaskType:    req.TaskType,
		UserID:      userID,
	}

	result := database.GetDB().Create(&task)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create task",
		})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// UpdateTask handles PUT /api/tasks/:id
// Updates a task owned by the authenticated user
func UpdateTask(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	// Check if task exists and belongs to user
	var existingTask models.Task
	result := database.GetDB().Where("id = ? AND user_id = ?", taskID, userID).First(&existingTask)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch task",
			})
		}
		return
	}

	// Parse update request
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Update fields if provided
	if req.Title != nil {
		existingTask.Title = *req.Title
	}
	if req.Description != nil {
		existingTask.Description = *req.Description
	}
	if req.Status != nil {
		existingTask.Status = *req.Status
	}
	if req.ProjectID != nil {
		existingTask.ProjectID = *req.ProjectID
	}
	if req.Assignee != nil {
		existingTask.AssigneeID = req.Assignee.ID
		existingTask.Assignee = *req.Assignee
	}
	if req.StartDate != nil {
		existingTask.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		existingTask.EndDate = *req.EndDate
	}
	if req.Effort != nil {
		existingTask.Effort = *req.Effort
	}
	if req.Priority != nil {
		existingTask.Priority = *req.Priority
	}
	if req.TaskType != nil {
		existingTask.TaskType = *req.TaskType
	}

	// Save updated task
	result = database.GetDB().Save(&existingTask)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update task",
		})
		return
	}

	c.JSON(http.StatusOK, existingTask)
}

// DeleteTask handles DELETE /api/tasks/:id
// Deletes a task owned by the authenticated user
func DeleteTask(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	// Check if task exists and belongs to user
	var task models.Task
	result := database.GetDB().Where("id = ? AND user_id = ?", taskID, userID).First(&task)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to fetch task",
			})
		}
		return
	}

	// Delete task
	result = database.GetDB().Delete(&task)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete task",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
		"id":      taskID,
	})
}