package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SkillHandler struct {
	catalog *service.CatalogService
	scanner *service.ScanService
	exec    *service.ExecService
}

func NewSkillHandler(catalog *service.CatalogService, scanner *service.ScanService, exec *service.ExecService) *SkillHandler {
	return &SkillHandler{catalog: catalog, scanner: scanner, exec: exec}
}

// Commands lists the executable commands declared in the skill's
// package.json (live parse with catalog fallback).
func (h *SkillHandler) Commands(c *gin.Context) {
	view, err := h.exec.Commands(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, view)
}

// Execs lists audited exec invocations, newest first. Query params:
// skill=<zid> filters to one skill; limit caps the page (default 20).
func (h *SkillHandler) Execs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	records, err := h.exec.ListExecRecords(c.Request.Context(), c.Query("skill"), limit)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, records)
}

// skillExecRequest is the JSON body of POST /api/skills/:zid/exec.
type skillExecRequest struct {
	Command        string   `json:"command"`
	Args           []string `json:"args"`
	Input          string   `json:"input"`
	Env            []string `json:"env"`
	AssumeYes      bool     `json:"assumeYes"`
	TimeoutSeconds int      `json:"timeoutSeconds"`
	Isolate        bool     `json:"isolate"`
	Pin            string   `json:"pin"`
	DryRun         bool     `json:"dryRun"`
}

// Exec runs a declared command of the skill and returns the structured
// result. Pre-check failures (unknown command, missing env, confirm gate,
// invalid input, unusable pin) surface as HTTP errors; a failing command
// itself is a normal 200 response carrying its exit code.
func (h *SkillHandler) Exec(c *gin.Context) {
	var req skillExecRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}
	if strings.TrimSpace(req.Command) == "" {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}
	for _, entry := range req.Env {
		if !strings.Contains(entry, "=") {
			writeServiceError(c, service.ErrInvalidInput)
			return
		}
	}

	result, err := h.exec.Exec(c.Request.Context(), &service.ExecRequest{
		SkillZid:        c.Param("zid"),
		Command:         req.Command,
		Args:            req.Args,
		InputJSON:       []byte(req.Input),
		Env:             req.Env,
		AssumeYes:       req.AssumeYes,
		TimeoutOverride: time.Duration(req.TimeoutSeconds) * time.Second,
		DryRun:          req.DryRun,
		Isolate:         req.Isolate,
		Pin:             req.Pin,
		Trigger:         "http",
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *SkillHandler) List(c *gin.Context) {
	var conflictFilter *bool
	if raw := strings.TrimSpace(c.Query("conflict")); raw != "" {
		parsed := strings.EqualFold(raw, "true") || raw == "1"
		conflictFilter = &parsed
	}
	grouped := false
	if raw := strings.TrimSpace(c.Query("grouped")); raw != "" {
		grouped = strings.EqualFold(raw, "true") || raw == "1"
	}
	filters := service.SkillListFilters{
		Query:    c.Query("q"),
		Provider: c.Query("provider"),
		Category: c.Query("category"),
		Tag:      c.Query("tag"),
		Status:   c.Query("status"),
		Conflict: conflictFilter,
		Sort:     c.Query("sort"),
		Grouped:  grouped,
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

func (h *SkillHandler) Attach(c *gin.Context) {
	var req service.SkillAttachInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeServiceError(c, service.ErrInvalidInput)
		return
	}

	result, err := h.catalog.AttachSkill(c.Request.Context(), c.Param("zid"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	jobs := make([]service.SkillAttachScanJob, 0, 2)
	if result.Mode == "move" {
		sourceJob, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), result.SourceProvider.Zid)
		if scanErr != nil {
			writeServiceError(c, scanErr)
			return
		}
		jobs = append(jobs, service.SkillAttachScanJob{ProviderZid: result.SourceProvider.Zid, Job: *sourceJob})
	}

	targetJob, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), result.TargetProvider.Zid)
	if scanErr != nil {
		writeServiceError(c, scanErr)
		return
	}
	jobs = append(jobs, service.SkillAttachScanJob{ProviderZid: result.TargetProvider.Zid, Job: *targetJob})

	result.Jobs = jobs
	response.OK(c, result)
}

func (h *SkillHandler) Delete(c *gin.Context) {
	var req service.SkillDeleteInput
	if c.Request.Body != nil {
		if err := c.ShouldBindJSON(&req); err != nil && !strings.Contains(strings.ToLower(err.Error()), "eof") {
			writeServiceError(c, service.ErrInvalidInput)
			return
		}
	}

	result, err := h.catalog.DeleteSkill(c.Request.Context(), c.Param("zid"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}

	job, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), result.Provider.Zid)
	if scanErr != nil {
		writeServiceError(c, scanErr)
		return
	}
	result.Job = job
	response.OK(c, result)
}

func (h *SkillHandler) Sync(c *gin.Context) {
	result, err := h.catalog.SyncSkill(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	job, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), result.Provider.Zid)
	if scanErr != nil {
		writeServiceError(c, scanErr)
		return
	}
	result.Job = job
	response.OK(c, result)
}

func (h *SkillHandler) SyncCopies(c *gin.Context) {
	result, err := h.catalog.SyncSkillCopies(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	providerZids := result.ScannedProviderZids
	if len(providerZids) == 0 {
		providerZids = []string{result.Provider.Zid}
	}
	for _, providerZid := range providerZids {
		job, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), providerZid)
		if scanErr != nil {
			continue
		}
		if providerZid == result.Provider.Zid {
			result.Job = job
		}
	}
	response.OK(c, result)
}

func (h *SkillHandler) RemoveRelation(c *gin.Context) {
	result, err := h.catalog.RemoveSkillRelation(c.Request.Context(), c.Param("zid"))
	if err != nil {
		writeServiceError(c, err)
		return
	}

	providerZids := result.ScannedProviderZids
	if len(providerZids) == 0 {
		providerZids = []string{result.Provider.Zid}
	}
	for _, providerZid := range providerZids {
		job, scanErr := h.scanner.ScanProviderByZid(c.Request.Context(), providerZid)
		if scanErr != nil {
			continue
		}
		if providerZid == result.Provider.Zid {
			result.Job = job
		}
	}
	response.OK(c, result)
}
