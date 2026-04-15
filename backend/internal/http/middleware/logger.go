package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestLogger logs HTTP request information
func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	httpLogger := logger.WithOptions(zap.WithCaller(false))

	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", clientIP),
		}

		if rawQuery != "" {
			fields = append(fields, zap.String("query", rawQuery))
		}

		if reqID, ok := c.Get("request_id"); ok {
			if id, ok := reqID.(string); ok {
				fields = append(fields, zap.String("request_id", id))
			}
		}

		if ownerZid, ok := c.Get("ownerZid"); ok {
			if zid, ok := ownerZid.(string); ok {
				fields = append(fields, zap.String("owner_zid", zid))
			}
		}

		if len(c.Errors) > 0 {
			errors := make([]string, 0, len(c.Errors))
			for _, item := range c.Errors {
				message := strings.TrimSpace(item.Error())
				if message != "" {
					errors = append(errors, message)
				}
			}
			if len(errors) > 0 {
				fields = append(fields,
					zap.Int("error_count", len(errors)),
					zap.Strings("errors", errors),
				)
			}
		}

		httpLogger.Info("http_request", fields...)
	}
}
