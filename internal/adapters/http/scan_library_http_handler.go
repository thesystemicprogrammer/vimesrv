package http

import (
	"github.com/gin-gonic/gin"
	"github.com/thesystemicprogrammer/vimesrv/internal/infrastructure/server"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared"
	"github.com/thesystemicprogrammer/vimesrv/internal/shared/logger"
	usecasejob "github.com/thesystemicprogrammer/vimesrv/internal/usecase/job"
)

type ScanLibraryHTTPHandler struct {
	enqueueJobUseCase *usecasejob.EnqueueJobUseCase
}

func NewScanLibraryHTTPHandler(enqueueJobUseCase *usecasejob.EnqueueJobUseCase) *ScanLibraryHTTPHandler {
	return &ScanLibraryHTTPHandler{
		enqueueJobUseCase: enqueueJobUseCase,
	}
}

// RegisterRoutes registers scan library routes on a router group (typically protected)
func (h *ScanLibraryHTTPHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/scanlib", h.Handle)
	logger.Debug().Msg("Scan library routes registered")
}

// Handle processes POST requests to trigger a library scan.
// It enqueues a scan_library job and returns 201 with the job ID.
func (h *ScanLibraryHTTPHandler) Handle(c *gin.Context) {
	// Create job input with no payload - uses config.Media.LibraryPath
	jobInput := usecasejob.EnqueueJobInput{
		Type:     shared.JobTypeScanLibrary,
		Payload:  nil,
		Priority: shared.JobPriorityLibraryScan,
	}

	// Enqueue the scan job
	jobID, err := h.enqueueJobUseCase.Execute(c.Request.Context(), jobInput)
	if err != nil {
		logger.Error().Err(err).Msg("failed to enqueue library scan job")
		c.JSON(500, server.ErrorResponse("ENQUEUE_FAILED", "Failed to enqueue scan job", err.Error()))
		return
	}

	logger.Info().Int64("ID", jobID).Str("type", shared.JobTypeScanLibrary).Msg("job enqueued successfully")

	// Return 201 Created with job ID
	c.JSON(201, server.SuccessResponse(map[string]interface{}{
		"job_id": jobID,
	}))
}
