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

// UpdateTaskStatusRequest represents a minimal request to change status
type UpdateTaskStatusRequest struct {
    Status models.TaskStatus `json:"status" binding:"required"`
}

func parseDateFlexible(dateStr string) (time.Time, bool) {
    if dateStr == "" {
        return time.Time{}, false
    }
    layouts := []string{
        "2006-01-02",    // ISO date
        "2 Jan 2006",    // e.g., 30 Oct 2025
        time.RFC3339,     // full RFC3339
        "02 Jan 2006",   // zero-padded day
    }
    for _, layout := range layouts {
        if t, err := time.Parse(layout, dateStr); err == nil {
            return t, true
        }
    }
    return time.Time{}, false
}

func calculateEffortDays(startDateStr, endDateStr string) int {
    start, okStart := parseDateFlexible(startDateStr)
    end, okEnd := parseDateFlexible(endDateStr)
    if !okStart || !okEnd {
        // Fallback to minimum effort 1 when dates invalid/missing
        return 1
    }
    // Normalize to midnight to avoid partial-day rounding issues
    start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
    end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
    if end.Before(start) {
        start, end = end, start
    }
    days := int(end.Sub(start).Hours() / 24)
    if days < 1 {
        return 1
    }
    return days
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

	// Enrich assignee field for response
	var users []models.User
	if err := database.GetDB().Find(&users).Error; err == nil {
		userByID := make(map[string]models.User, len(users))
		for _, u := range users {
			userByID[u.ID] = u
		}

        for i := range tasks {
            if u, ok := userByID[tasks[i].AssigneeID]; ok {
                tasks[i].Assignee = models.Assignee{
                    ID:   u.ID,
                    Name: u.Username,
                }
            }
        }
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

    // Compute effort based on dates; ignore client-provided effort
    effort := calculateEffortDays(req.StartDate, req.EndDate)

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

    // No avatar handling

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
    // Recalculate effort if either date was provided in the update; otherwise leave as-is
    if req.StartDate != nil || req.EndDate != nil {
        existingTask.Effort = calculateEffortDays(existingTask.StartDate, existingTask.EndDate)
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

	// Enrich assignee in response
    if existingTask.AssigneeID != "" {
        var u models.User
        if err := database.GetDB().Where("id = ?", existingTask.AssigneeID).First(&u).Error; err == nil {
            existingTask.Assignee = models.Assignee{ ID: u.ID, Name: u.Username }
        }
    }

	c.JSON(http.StatusOK, existingTask)
}

// GetTaskByID handles GET /api/tasks/:id
// Returns a single task owned by the authenticated user
func GetTaskByID(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "User ID not found in token",
        })
        return
    }

    taskID := c.Param("id")
    if taskID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
        return
    }

    var task models.Task
    result := database.GetDB().Where("id = ? AND user_id = ?", taskID, userID).First(&task)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
        }
        return
    }

    // Enrich assignee
    if task.AssigneeID != "" {
        var u models.User
        if err := database.GetDB().Where("id = ?", task.AssigneeID).First(&u).Error; err == nil {
            task.Assignee = models.Assignee{ID: u.ID, Name: u.Username}
        }
    }

    c.JSON(http.StatusOK, task)
}

// UpdateTaskStatus handles PATCH /api/tasks/:id/status
// Updates only the status of a task owned by the authenticated user
func UpdateTaskStatus(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error": "User ID not found in token",
        })
        return
    }

    taskID := c.Param("id")
    if taskID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Task ID is required"})
        return
    }

    var req UpdateTaskStatusRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    var task models.Task
    result := database.GetDB().Where("id = ? AND user_id = ?", taskID, userID).First(&task)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch task"})
        }
        return
    }

    // Explicitly update only the status column to ensure persistence
    task.Status = req.Status
    if err := database.GetDB().Model(&task).Update("status", req.Status).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
        return
    }

    // Enrich assignee in response
    if task.AssigneeID != "" {
        var u models.User
        if err := database.GetDB().Where("id = ?", task.AssigneeID).First(&u).Error; err == nil {
            task.Assignee = models.Assignee{ID: u.ID, Name: u.Username}
        }
    }

    c.JSON(http.StatusOK, task)
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