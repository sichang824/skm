package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	catalog *service.CatalogService
}

func NewDashboardHandler(catalog *service.CatalogService) *DashboardHandler {
	return &DashboardHandler{catalog: catalog}
}

func (h *DashboardHandler) Get(c *gin.Context) {
	data, err := h.catalog.Dashboard(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, data)
}
