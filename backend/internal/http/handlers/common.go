package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	_ = c.Error(err)

	// Exec pre-check failures reject the invocation before any process
	// starts. The confirm gate gets its own status so the UI can prompt the
	// user and retry with assumeYes.
	var confirmRequired *service.ExecConfirmRequired
	if errors.As(err, &confirmRequired) {
		response.Error(c, http.StatusConflict, http.StatusConflict, err.Error())
		return
	}
	var commandNotFound *service.ExecCommandNotFound
	var missingEnv *service.ExecMissingEnv
	var inputInvalid *service.ExecInputInvalid
	var rootMissing *service.ExecRootMissing
	var pinInvalid *service.ExecPinInvalid
	var pinUnavailable *service.ExecPinUnavailable
	switch {
	case errors.Is(err, service.ErrProviderNotFound), errors.Is(err, service.ErrSkillNotFound), errors.Is(err, service.ErrScanJobNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrBinaryFile):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.As(err, &commandNotFound), errors.As(err, &missingEnv),
		errors.As(err, &inputInvalid), errors.As(err, &rootMissing),
		errors.As(err, &pinInvalid), errors.As(err, &pinUnavailable),
		errors.Is(err, service.ErrExecManifestMissing), errors.Is(err, service.ErrExecSetupMissing):
		response.Error(c, http.StatusUnprocessableEntity, http.StatusUnprocessableEntity, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
	}
}
