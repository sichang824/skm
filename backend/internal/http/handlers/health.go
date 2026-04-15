package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	version string
}

func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{version: version}
}

// Healthz returns health status
func (h *HealthHandler) Healthz(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// Version returns version information
func (h *HealthHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": h.version,
		"status":  "running",
	})
}
