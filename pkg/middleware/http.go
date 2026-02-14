package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"time"
)

func WithTimeout(ts time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), ts)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func WithSource(source string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("source", source)
		ctx := context.WithValue(c.Request.Context(), "source", source)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func WithRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}
