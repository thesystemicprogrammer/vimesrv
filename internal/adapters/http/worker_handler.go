package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/config"
	workeruc "github.com/thesystemicprogrammer/vimesrv/internal/usecase/worker"
)

// WorkerHandler handles worker-related HTTP requests
type WorkerHandler struct {
	config           *config.WorkerConfig
	registerUC       *workeruc.RegisterWorkerUseCase
	heartbeatUC      *workeruc.HeartbeatUseCase
	claimJobUC       *workeruc.ClaimJobForWorkerUseCase
	completeJobUC    *workeruc.CompleteWorkerJobUseCase
	failJobUC        *workeruc.FailWorkerJobUseCase
	reportProgressUC *workeruc.ReportProgressUseCase
}

// NewWorkerHandler creates a new WorkerHandler
func NewWorkerHandler(
	cfg *config.WorkerConfig,
	registerUC *workeruc.RegisterWorkerUseCase,
	heartbeatUC *workeruc.HeartbeatUseCase,
	claimJobUC *workeruc.ClaimJobForWorkerUseCase,
	completeJobUC *workeruc.CompleteWorkerJobUseCase,
	failJobUC *workeruc.FailWorkerJobUseCase,
	reportProgressUC *workeruc.ReportProgressUseCase,
) *WorkerHandler {
	return &WorkerHandler{
		config:           cfg,
		registerUC:       registerUC,
		heartbeatUC:      heartbeatUC,
		claimJobUC:       claimJobUC,
		completeJobUC:    completeJobUC,
		failJobUC:        failJobUC,
		reportProgressUC: reportProgressUC,
	}
}

// RegisterRoutes registers worker API routes
func (h *WorkerHandler) RegisterRoutes(router *gin.RouterGroup) {
	worker := router.Group("/worker")
	worker.Use(h.workerAuthMiddleware())
	{
		worker.POST("/register", h.Register)
		worker.POST("/heartbeat", h.Heartbeat)

		jobs := worker.Group("/jobs")
		{
			jobs.POST("/claim", h.ClaimJob)
			jobs.POST("/:id/progress", h.ReportProgress)
			jobs.POST("/:id/complete", h.CompleteJob)
			jobs.POST("/:id/fail", h.FailJob)
		}
	}
}

// workerAuthMiddleware validates the worker auth token
func (h *WorkerHandler) workerAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if worker API is enabled
		if !h.config.Enabled {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Worker API not enabled",
				},
			})
			c.Abort()
			return
		}

		// Validate Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Missing or invalid Authorization header",
				},
			})
			c.Abort()
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != h.config.AuthToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid auth token",
				},
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RegisterRequest is the request body for POST /worker/register
type RegisterRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
}

// Register handles POST /api/v1/worker/register
func (h *WorkerHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	err := h.registerUC.Execute(c.Request.Context(), workeruc.RegisterWorkerInput{
		WorkerID: req.WorkerID,
		Name:     req.Name,
		Capacity: req.Capacity,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"registered": true,
		"message":    "Worker registered successfully",
	})
}

// HeartbeatRequest is the request body for POST /worker/heartbeat
type HeartbeatRequest struct {
	WorkerID   string `json:"worker_id" binding:"required"`
	Name       string `json:"name"`
	ActiveJobs int    `json:"active_jobs"`
	Capacity   int    `json:"capacity"`
}

// Heartbeat handles POST /api/v1/worker/heartbeat
func (h *WorkerHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	output, err := h.heartbeatUC.Execute(c.Request.Context(), workeruc.HeartbeatInput{
		WorkerID:   req.WorkerID,
		Name:       req.Name,
		ActiveJobs: req.ActiveJobs,
		Capacity:   req.Capacity,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":          output.OK,
		"server_time": output.ServerTime,
		"queued_jobs": output.QueuedJobs,
	})
}

// ClaimJobRequest is the request body for POST /worker/jobs/claim
type ClaimJobRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
}

// ClaimJob handles POST /api/v1/worker/jobs/claim
func (h *WorkerHandler) ClaimJob(c *gin.Context) {
	var req ClaimJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	job, err := h.claimJobUC.Execute(c.Request.Context(), req.WorkerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Return job or null if no jobs available
	c.JSON(http.StatusOK, gin.H{
		"job": job, // nil becomes null in JSON
	})
}

// ProgressRequest is the request body for POST /worker/jobs/:id/progress
type ProgressRequest struct {
	WorkerID   string  `json:"worker_id" binding:"required"`
	Percentage float64 `json:"percentage"`
	Speed      string  `json:"speed"`
	ETASeconds int     `json:"eta_seconds"`
}

// ReportProgress handles POST /api/v1/worker/jobs/:id/progress
func (h *WorkerHandler) ReportProgress(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid job ID",
			},
		})
		return
	}

	var req ProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	err = h.reportProgressUC.Execute(c.Request.Context(), workeruc.ReportProgressInput{
		JobID:      jobID,
		WorkerID:   req.WorkerID,
		Percentage: req.Percentage,
		Speed:      req.Speed,
		ETASeconds: req.ETASeconds,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "JOB_NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok": true,
	})
}

// CompleteJobRequest is the request body for POST /worker/jobs/:id/complete
type CompleteJobRequest struct {
	WorkerID     string   `json:"worker_id" binding:"required"`
	SegmentCount int      `json:"segment_count"`
	OutputFiles  []string `json:"output_files"`
}

// CompleteJob handles POST /api/v1/worker/jobs/:id/complete
func (h *WorkerHandler) CompleteJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid job ID",
			},
		})
		return
	}

	var req CompleteJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	err = h.completeJobUC.Execute(c.Request.Context(), workeruc.CompleteJobInput{
		JobID:        jobID,
		WorkerID:     req.WorkerID,
		SegmentCount: req.SegmentCount,
		OutputFiles:  req.OutputFiles,
	})
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "validation failed") {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{
					"code":    "VALIDATION_FAILED",
					"message": err.Error(),
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "Job completed successfully",
	})
}

// FailJobRequest is the request body for POST /worker/jobs/:id/fail
type FailJobRequest struct {
	WorkerID string `json:"worker_id" binding:"required"`
	Error    string `json:"error" binding:"required"`
	Retry    bool   `json:"retry"`
}

// FailJob handles POST /api/v1/worker/jobs/:id/fail
func (h *WorkerHandler) FailJob(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid job ID",
			},
		})
		return
	}

	var req FailJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	err = h.failJobUC.Execute(c.Request.Context(), workeruc.FailJobInput{
		JobID:    jobID,
		WorkerID: req.WorkerID,
		Error:    req.Error,
		Retry:    req.Retry,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	message := "Job marked as failed"
	if req.Retry {
		message = "Job marked as failed, will retry"
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": message,
	})
}
