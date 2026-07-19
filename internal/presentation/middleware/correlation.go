package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const CorrelationIDHeader = "X-Correlation-ID"
const CorrelationIDKey = "correlation_id"

// CorrelationID ensures every request carries a correlation ID, generating
// one when the caller didn't supply it, and echoes it back on the response.
func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(CorrelationIDHeader)
		if id == "" {
			id = uuid.New().String()
		}
		c.Set(CorrelationIDKey, id)
		c.Header(CorrelationIDHeader, id)
		c.Next()
	}
}

func GetCorrelationID(c *gin.Context) string {
	if v, ok := c.Get(CorrelationIDKey); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
