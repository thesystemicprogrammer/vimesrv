package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
)

// JobDTO represents a job in the API response
type JobDTO struct {
	ID          int64       `json:"id"`
	Type        string      `json:"type"`
	Status      string      `json:"status"`
	Payload     interface{} `json:"payload,omitempty"`
	Priority    int         `json:"priority"`
	Attempts    int         `json:"attempts"`
	MaxAttempts int         `json:"max_attempts"`
	LastError   *string     `json:"last_error,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	StartedAt   *time.Time  `json:"started_at,omitempty"`
	FinishedAt  *time.Time  `json:"finished_at,omitempty"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// JobListResponse is the response for listing jobs
type JobListResponse struct {
	Jobs  []JobDTO `json:"jobs"`
	Total int      `json:"total"`
}

// JobHandler handles job-related HTTP requests
type JobHandler struct {
	listJobsUseCase *usecasejob.ListJobsUseCase
}

// NewJobHandler creates a new JobHandler
func NewJobHandler(listJobsUseCase *usecasejob.ListJobsUseCase) *JobHandler {
	return &JobHandler{
		listJobsUseCase: listJobsUseCase,
	}
}

// ListJobs handles GET /api/v1/jobs
// Query params:
//   - status: comma-separated list of statuses (queued,running,succeeded,dead)
//   - type: comma-separated list of job types
//   - include_old: if "true", include jobs older than 24h
func (h *JobHandler) ListJobs(c *gin.Context) {
	// Parse query parameters
	input := usecasejob.ListJobsInput{
		Limit: 100, // Default limit
	}

	// Parse status filter
	if statusParam := c.Query("status"); statusParam != "" {
		input.Statuses = strings.Split(statusParam, ",")
	}

	// Parse type filter
	if typeParam := c.Query("type"); typeParam != "" {
		input.Types = strings.Split(typeParam, ",")
	}

	// Parse include_old flag
	if c.Query("include_old") == "true" {
		input.IncludeOlderThan24h = true
	}

	// Execute use case
	output, err := h.listJobsUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to list jobs",
			},
		})
		return
	}

	// Convert to DTOs
	jobs := make([]JobDTO, 0, len(output.Jobs))
	for _, job := range output.Jobs {
		dto := JobDTO{
			ID:          job.ID,
			Type:        job.Type,
			Status:      string(job.Status),
			Priority:    job.Priority,
			Attempts:    job.Attempts,
			MaxAttempts: job.MaxAttempts,
			CreatedAt:   job.CreatedAt,
			UpdatedAt:   job.UpdatedAt,
		}

		// Parse payload as JSON if present
		if len(job.Payload) > 0 {
			var payload interface{}
			if err := json.Unmarshal(job.Payload, &payload); err == nil {
				dto.Payload = payload
			}
		}

		// Handle nullable fields
		if job.LastError.Valid {
			dto.LastError = &job.LastError.String
		}
		if job.StartedAt.Valid {
			dto.StartedAt = &job.StartedAt.Time
		}
		if job.FinishedAt.Valid {
			dto.FinishedAt = &job.FinishedAt.Time
		}

		jobs = append(jobs, dto)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": JobListResponse{
			Jobs:  jobs,
			Total: output.Total,
		},
	})
}

// RegisterRoutes registers job-related routes with manager middleware
func (h *JobHandler) RegisterRoutes(router *gin.RouterGroup) {
	jobs := router.Group("/jobs")
	jobs.Use(h.requireManager())
	{
		jobs.GET("", h.ListJobs)
	}
}

// requireManager middleware checks if the user has admin or manager role
func (h *JobHandler) requireManager() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Access denied",
				"",
			))
			return
		}

		roleStr := role.(string)
		if roleStr != string(shared.RoleAdmin) && roleStr != string(shared.RoleManager) {
			c.AbortWithStatusJSON(http.StatusForbidden, server.ErrorResponse(
				"FORBIDDEN",
				"Admin or manager access required",
				"",
			))
			return
		}

		c.Next()
	}
}
