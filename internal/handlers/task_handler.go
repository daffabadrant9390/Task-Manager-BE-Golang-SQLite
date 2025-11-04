package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"task-management-api/internal/database"
	"task-management-api/internal/models"
	"task-management-api/internal/realtime"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// CreateTaskRequest represents the request payload for creating a task
type CreateTaskRequest struct {
	Title       string              `json:"title" binding:"required"`
	Description string              `json:"description" binding:"required"`
	Status      models.TaskStatus   `json:"status"`
	ProjectID   string              `json:"projectId"`
	Assignee    models.Assignee     `json:"assignee" binding:"required"`
	StartDate   string              `json:"startDate" binding:"required"`
	EndDate     string              `json:"endDate" binding:"required"`
	Effort      int                 `json:"effort"`
	Priority    models.TaskPriority `json:"priority"`
	TaskType    models.TaskType     `json:"taskType" binding:"required"`
}

// UpdateTaskRequest represents the request payload for updating a task
type UpdateTaskRequest struct {
	Title       *string              `json:"title"`
	Description *string              `json:"description"`
	Status      *models.TaskStatus   `json:"status"`
	ProjectID   *string              `json:"projectId"`
	Assignee    *models.Assignee     `json:"assignee"`
	StartDate   *string              `json:"startDate"`
	EndDate     *string              `json:"endDate"`
	Effort      *int                 `json:"effort"`
	Priority    *models.TaskPriority `json:"priority"`
	TaskType    *models.TaskType     `json:"taskType"`
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
		"2006-01-02",  // ISO date
		"2 Jan 2006",  // e.g., 30 Oct 2025
		time.RFC3339,  // full RFC3339
		"02 Jan 2006", // zero-padded day
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

/*
*
GetTasks handles GET /api/tasks
Returns all tasks (team-wide) for authenticated users.
Optional query param: userId to filter tasks created by a specific user.
*/
func GetTasks(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in token",
		})
		return
	}

	// Query params: page (default 1), limit (default 5), sort (asc|desc on created_at, default desc)
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "5")
	sortParam := strings.ToLower(c.DefaultQuery("sort", "desc"))
	filterUserID := c.Query("userId") // optional: filter by creator

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 5
	}
	if limit > 100 {
		limit = 100
	}

	offset := (page - 1) * limit

	order := "created_at desc"
	if sortParam == "asc" {
		order = "created_at asc"
	}

	// Build base query (team-wide); optionally filter by specified userId
	db := database.GetDB()
	query := db.Model(&models.Task{})
	if filterUserID != "" {
		query = query.Where("user_id = ?", filterUserID)
	}

	// Total count (without pagination)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to count tasks",
		})
		return
	}

	// Fetch paginated tasks with sorting
	var tasks []models.Task
	result := query.Session(&gorm.Session{}).Order(order).Limit(limit).Offset(offset).Find(&tasks)
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
		"count": len(tasks), // number of items in this page
		"total": total,      // total tasks (all pages) for current filter
		"page":  page,
		"limit": limit,
		"sort":  sortParam,
	})
}

/*
*
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

	// Validate and normalize project linkage based on task type
	projectID := strings.TrimSpace(req.ProjectID)
	switch req.TaskType {
	case models.TypeStory:
		// Level 1: must NOT be linked; treat empty as null and enforce empty
		projectID = ""
	case models.TypeDefect, models.TypeSubtask:
		// Level 2: must reference an existing Story as parent via projectId
		if projectID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "projectId is required for subtask/defect and must reference a story id"})
			return
		}
		// Validate parent exists and is a story owned by the same team (no user ownership requirement for parent beyond visibility)
		var parent models.Task
		if err := database.GetDB().Where("id = ? AND task_type = ?", projectID, models.TypeStory).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid projectId: parent story not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate projectId"})
			}
			return
		}
	default:
		// Unknown type guard
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid taskType"})
		return
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

	// Broadcast event to the authenticated user's channels
	evt := map[string]any{
		"type":    "task_created",
		"taskId":  task.ID,
		"userId":  userID,
		"version": 1,
	}
	if bytes, err := json.Marshal(evt); err == nil {
		realtime.GetHub().Broadcast(userID, bytes)
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

	// Enforce projectId invariants based on (possibly updated) type
	// Rules: story => projectId must be empty; subtask/defect => projectId required and must reference existing story
	if existingTask.TaskType == models.TypeStory {
		existingTask.ProjectID = ""
	} else if existingTask.TaskType == models.TypeDefect || existingTask.TaskType == models.TypeSubtask {
		if strings.TrimSpace(existingTask.ProjectID) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "projectId is required for subtask/defect and must reference a story id"})
			return
		}
		var parent models.Task
		if err := database.GetDB().Where("id = ? AND task_type = ?", existingTask.ProjectID, models.TypeStory).First(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid projectId: parent story not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate projectId"})
			}
			return
		}
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
			existingTask.Assignee = models.Assignee{ID: u.ID, Name: u.Username}
		}
	}

	// Broadcast update event
	evt := map[string]any{
		"type":    "task_updated",
		"taskId":  existingTask.ID,
		"userId":  userID,
		"version": 1,
	}
	if bytes, err := json.Marshal(evt); err == nil {
		realtime.GetHub().Broadcast(userID, bytes)
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

	// Broadcast status change
	evt := map[string]any{
		"type":    "task_status_changed",
		"taskId":  task.ID,
		"userId":  userID,
		"version": 1,
	}
	if bytes, err := json.Marshal(evt); err == nil {
		realtime.GetHub().Broadcast(userID, bytes)
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

	// Broadcast deletion
	evt := map[string]any{
		"type":    "task_deleted",
		"taskId":  taskID,
		"userId":  userID,
		"version": 1,
	}
	if bytes, err := json.Marshal(evt); err == nil {
		realtime.GetHub().Broadcast(userID, bytes)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Task deleted successfully",
		"id":      taskID,
	})
}

// GetStatsByUser handles GET /api/stats/:userid
// Returns counts of tasks by status (todo, inProgress, done) where the assignee matches :userid
func GetStatsByUser(c *gin.Context) {
	// Ensure request is authenticated
	authUserID := c.GetString("user_id")
	if authUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	targetUserID := c.Param("userid")
	if strings.TrimSpace(targetUserID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}

	db := database.GetDB()

	type row struct {
		Status string
		Count  int64
	}

	var rows []row
	if err := db.Model(&models.Task{}).
		Select("status, COUNT(*) as count").
		Where("assignee_id = ?", targetUserID).
		Group("status").
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to compute stats"})
		return
	}

	// Initialize with zeros
	counts := map[string]int64{
		string(models.StatusTodo):       0,
		string(models.StatusInProgress): 0,
		string(models.StatusDone):       0,
	}
	var total int64 = 0
	for _, r := range rows {
		counts[r.Status] = r.Count
		total += r.Count
	}

	c.JSON(http.StatusOK, gin.H{
		"todo":       counts[string(models.StatusTodo)],
		"inProgress": counts[string(models.StatusInProgress)],
		"done":       counts[string(models.StatusDone)],
		"total":      total,
	})
}
