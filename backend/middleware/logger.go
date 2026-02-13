package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		// Log attributes
		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		}

		if query != "" {
			attrs = append(attrs, slog.String("query", query))
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}

		// Log level
		status := c.Writer.Status()
		switch {
		case status >= 500:
			slog.LogAttrs(c, slog.LevelError, "Server Error", attrs...)
		case status >= 400:
			slog.LogAttrs(c, slog.LevelWarn, "Client Error", attrs...)
		default:
			slog.LogAttrs(c, slog.LevelInfo, "Request", attrs...)
		}
	}
}