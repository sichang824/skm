package handlers

import (
	"backend-go/internal/http/response"
	"backend-go/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrProviderNotFound), errors.Is(err, service.ErrSkillNotFound), errors.Is(err, service.ErrScanJobNotFound):
		response.Error(c, http.StatusNotFound, http.StatusNotFound, err.Error())
	case errors.Is(err, service.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrBinaryFile):
		response.Error(c, http.StatusBadRequest, http.StatusBadRequest, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
	}
}
