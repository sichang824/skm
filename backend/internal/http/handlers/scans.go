package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"

	"github.com/gin-gonic/gin"
)

type ScanHandler struct {
	catalog *service.CatalogService
	scanner *service.ScanService
}

func NewScanHandler(catalog *service.CatalogService, scanner *service.ScanService) *ScanHandler {
	return &ScanHandler{catalog: catalog, scanner: scanner}
}

func (h *ScanHandler) ScanAll(c *gin.Context) {
	result, err := h.scanner.ScanAllProviders(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *ScanHandler) ListJobs(c *gin.Context) {
	jobs, err := h.catalog.ListScanJobs(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, jobs)
}

func (h *ScanHandler) GetJob(c *gin.Context) {
	job, issues, err := h.catalog.GetScanJob(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"job": job, "issues": issues})
}

func (h *ScanHandler) ListIssues(c *gin.Context) {
	issues, err := h.catalog.ListIssues(c.Request.Context(), service.IssueListFilters{
		View:     c.Query("view"),
		Provider: c.Query("provider"),
		Severity: c.Query("severity"),
		Code:     c.Query("code"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, issues)
}

func (h *ScanHandler) ListConflicts(c *gin.Context) {
	conflicts, err := h.catalog.ListConflicts(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, conflicts)
}
