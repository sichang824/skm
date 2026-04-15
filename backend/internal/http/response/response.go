package response

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(200, APIResponse{Code: 0, Message: "ok", Data: data})
}

func Error(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, APIResponse{Code: code, Message: msg})
}

func ErrorWithData(c *gin.Context, httpStatus int, code int, msg string, data any) {
	c.JSON(httpStatus, APIResponse{Code: code, Message: msg, Data: data})
}
