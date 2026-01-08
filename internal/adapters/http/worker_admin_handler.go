package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/domain"
	workeruc "github.com/thesystemicprogrammer/vimesrv/internal/usecase/worker"
)

// WorkerReconfigurer is an interface for triggering worker reconfiguration
type WorkerReconfigurer interface {
	// Reconfigure reloads worker count and configs from database and applies changes
	Reconfigure(ctx context.Context) error
	// ReloadWorkerConfigs just reloads configs without changing worker count
	ReloadWorkerConfigs(ctx context.Context) error
}

// WorkerAdminHandler handles worker administration HTTP requests
type WorkerAdminHandler struct {
	listWorkerConfigsUC   *workeruc.ListWorkerConfigsUseCase
	getLocalWorkerCountUC *workeruc.GetLocalWorkerCountUseCase
	setLocalWorkerCountUC *workeruc.SetLocalWorkerCountUseCase
	getWorkerConfigUC     *workeruc.GetWorkerConfigUseCase
	updateWorkerConfigUC  *workeruc.UpdateWorkerConfigUseCase
	deleteWorkerConfigUC  *workeruc.DeleteWorkerConfigUseCase
	reconfigurer          WorkerReconfigurer
}

// NewWorkerAdminHandler creates a new WorkerAdminHandler
func NewWorkerAdminHandler(
	listWorkerConfigsUC *workeruc.ListWorkerConfigsUseCase,
	getLocalWorkerCountUC *workeruc.GetLocalWorkerCountUseCase,
	setLocalWorkerCountUC *workeruc.SetLocalWorkerCountUseCase,
	getWorkerConfigUC *workeruc.GetWorkerConfigUseCase,
	updateWorkerConfigUC *workeruc.UpdateWorkerConfigUseCase,
	deleteWorkerConfigUC *workeruc.DeleteWorkerConfigUseCase,
	reconfigurer WorkerReconfigurer,
) *WorkerAdminHandler {
	return &WorkerAdminHandler{
		listWorkerConfigsUC:   listWorkerConfigsUC,
		getLocalWorkerCountUC: getLocalWorkerCountUC,
		setLocalWorkerCountUC: setLocalWorkerCountUC,
		getWorkerConfigUC:     getWorkerConfigUC,
		updateWorkerConfigUC:  updateWorkerConfigUC,
		deleteWorkerConfigUC:  deleteWorkerConfigUC,
		reconfigurer:          reconfigurer,
	}
}

// RegisterRoutes registers worker admin routes (requires admin auth)
func (h *WorkerAdminHandler) RegisterRoutes(router *gin.RouterGroup) {
	admin := router.Group("/admin/workers")
	{
		// List all worker configs
		admin.GET("", h.ListWorkerConfigs)

		// Local worker count management
		admin.GET("/local/count", h.GetLocalWorkerCount)
		admin.PUT("/local/count", h.SetLocalWorkerCount)

		// Individual worker config management
		admin.GET("/:name", h.GetWorkerConfig)
		admin.PUT("/:name", h.UpdateWorkerConfig)
		admin.DELETE("/:name", h.DeleteWorkerConfig)
	}
}

// ListWorkerConfigs handles GET /api/v1/admin/workers
func (h *WorkerAdminHandler) ListWorkerConfigs(c *gin.Context) {
	output, err := h.listWorkerConfigsUC.Execute(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, output)
}

// GetLocalWorkerCount handles GET /api/v1/admin/workers/local/count
func (h *WorkerAdminHandler) GetLocalWorkerCount(c *gin.Context) {
	count, err := h.getLocalWorkerCountUC.Execute(c.Request.Context())
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
		"count": count,
	})
}

// SetLocalWorkerCountRequest is the request body for PUT /admin/workers/local/count
type SetLocalWorkerCountRequest struct {
	Count int `json:"count" binding:"required,min=1,max=16"`
}

// SetLocalWorkerCount handles PUT /api/v1/admin/workers/local/count
func (h *WorkerAdminHandler) SetLocalWorkerCount(c *gin.Context) {
	var req SetLocalWorkerCountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	err := h.setLocalWorkerCountUC.Execute(c.Request.Context(), workeruc.SetLocalWorkerCountInput{
		Count: req.Count,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": err.Error(),
			},
		})
		return
	}

	// Trigger reconfiguration to apply changes immediately
	if h.reconfigurer != nil {
		if err := h.reconfigurer.Reconfigure(c.Request.Context()); err != nil {
			// Log but don't fail the request - the database was updated successfully
			c.JSON(http.StatusOK, gin.H{
				"count":   req.Count,
				"message": "Worker count updated, but reconfiguration failed: " + err.Error(),
				"warning": "Changes may require server restart to take full effect",
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   req.Count,
		"message": "Local worker count updated and applied",
	})
}

// GetWorkerConfig handles GET /api/v1/admin/workers/:name
func (h *WorkerAdminHandler) GetWorkerConfig(c *gin.Context) {
	name := c.Param("name")

	config, err := h.getWorkerConfigUC.Execute(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, workerConfigToResponse(config))
}

// UpdateWorkerConfigRequest is the request body for PUT /admin/workers/:name
type UpdateWorkerConfigRequest struct {
	AcceptsVideo *bool `json:"accepts_video"`
	AcceptsAudio *bool `json:"accepts_audio"`
}

// UpdateWorkerConfig handles PUT /api/v1/admin/workers/:name
func (h *WorkerAdminHandler) UpdateWorkerConfig(c *gin.Context) {
	name := c.Param("name")

	var req UpdateWorkerConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BAD_REQUEST",
				"message": "Invalid request: " + err.Error(),
			},
		})
		return
	}

	config, err := h.updateWorkerConfigUC.Execute(c.Request.Context(), workeruc.UpdateWorkerConfigInput{
		Name:         name,
		AcceptsVideo: req.AcceptsVideo,
		AcceptsAudio: req.AcceptsAudio,
	})
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"code":    "ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	// Reload worker configs to apply changes immediately
	if h.reconfigurer != nil {
		_ = h.reconfigurer.ReloadWorkerConfigs(c.Request.Context())
	}

	c.JSON(http.StatusOK, workerConfigToResponse(config))
}

// DeleteWorkerConfig handles DELETE /api/v1/admin/workers/:name
func (h *WorkerAdminHandler) DeleteWorkerConfig(c *gin.Context) {
	name := c.Param("name")

	err := h.deleteWorkerConfigUC.Execute(c.Request.Context(), name)
	if err != nil {
		statusCode := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{
			"error": gin.H{
				"code":    "ERROR",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Worker config deleted",
	})
}

// WorkerConfigResponse is the JSON response for a worker config
type WorkerConfigResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	WorkerType   string `json:"worker_type"`
	AcceptsVideo bool   `json:"accepts_video"`
	AcceptsAudio bool   `json:"accepts_audio"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

func workerConfigToResponse(config *domain.WorkerConfig) WorkerConfigResponse {
	return WorkerConfigResponse{
		ID:           config.ID,
		Name:         config.Name,
		WorkerType:   string(config.WorkerType),
		AcceptsVideo: config.AcceptsVideo,
		AcceptsAudio: config.AcceptsAudio,
		CreatedAt:    config.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    config.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
