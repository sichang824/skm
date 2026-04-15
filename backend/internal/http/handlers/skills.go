package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"
	"strings"

	"github.com/gin-gonic/gin"
)

type SkillHandler struct {
	catalog *service.CatalogService
}

func NewSkillHandler(catalog *service.CatalogService) *SkillHandler {
	return &SkillHandler{catalog: catalog}
}

func (h *SkillHandler) List(c *gin.Context) {
	var conflictFilter *bool
	if raw := strings.TrimSpace(c.Query("conflict")); raw != "" {
		parsed := strings.EqualFold(raw, "true") || raw == "1"
		conflictFilter = &parsed
	}
	filters := service.SkillListFilters{
		Query:    c.Query("q"),
		Provider: c.Query("provider"),
		Category: c.Query("category"),
		Tag:      c.Query("tag"),
		Status:   c.Query("status"),
		Conflict: conflictFilter,
		Sort:     c.Query("sort"),
	}
	skills, err := h.catalog.ListSkills(c.Request.Context(), filters)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, skills)
}

func (h *SkillHandler) Get(c *gin.Context) {
	skill, err := h.catalog.GetSkill(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, skill)
}

func (h *SkillHandler) Files(c *gin.Context) {
	files, err := h.catalog.GetSkillFiles(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, files)
}

func (h *SkillHandler) FileContent(c *gin.Context) {
	content, err := h.catalog.GetSkillFileContent(c.Request.Context(), c.Param("zid"), c.Query("path"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, content)
}
