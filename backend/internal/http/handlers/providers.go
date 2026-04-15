package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

type ProviderHandler struct {
	catalog *service.CatalogService
	scanner *service.ScanService
}

func NewProviderHandler(catalog *service.CatalogService, scanner *service.ScanService) *ProviderHandler {
	return &ProviderHandler{catalog: catalog, scanner: scanner}
}

func (h *ProviderHandler) List(c *gin.Context) {
	providers, err := h.catalog.ListProviders(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, providers)
}

func (h *ProviderHandler) Get(c *gin.Context) {
	provider, err := h.catalog.GetProvider(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, provider)
}

func (h *ProviderHandler) Create(c *gin.Context) {
	var req service.ProviderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}
	provider, err := h.catalog.CreateProvider(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, provider)
}

func (h *ProviderHandler) Update(c *gin.Context) {
	var req service.ProviderInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}
	provider, err := h.catalog.UpdateProvider(c.Request.Context(), c.Param("zid"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, provider)
}

func (h *ProviderHandler) Delete(c *gin.Context) {
	if err := h.catalog.DeleteProvider(c.Request.Context(), c.Param("zid")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *ProviderHandler) Scan(c *gin.Context) {
	job, err := h.scanner.ScanProviderByZid(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, job)
}
