package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ObservabilityHandler struct{}

func NewObservabilityHandler() *ObservabilityHandler {
	return &ObservabilityHandler{}
}

func (h *ObservabilityHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler())) // gin.WrapH(promhttp.Handler()))
}

func (h *ObservabilityHandler) PrometheusMetrics(c *gin.Context) {
	gin.WrapH(promhttp.Handler())
}
