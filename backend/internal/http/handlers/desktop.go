package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

type DesktopHandler struct {
	desktop *service.DesktopService
}

func NewDesktopHandler(desktop *service.DesktopService) *DesktopHandler {
	return &DesktopHandler{desktop: desktop}
}

func (h *DesktopHandler) CLIStatus(c *gin.Context) {
	status, err := h.desktop.CLIStatus()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, status)
}

func (h *DesktopHandler) InstallCLI(c *gin.Context) {
	result, err := h.desktop.InstallCLI()
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *DesktopHandler) RevealInFinder(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}
	result, err := h.desktop.RevealInFinder(req.Path)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}
