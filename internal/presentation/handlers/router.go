package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/gabrielAnFran/pos-production-service/internal/presentation/middleware"
)

func NewRouter(exec *ExecutionHandler, health *HealthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CorrelationID())

	r.GET("/healthz", health.Healthz)
	r.GET("/readyz", health.Readyz)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/executions/:os_id", exec.Get)
		v1.PATCH("/executions/:os_id", exec.UpdateStatus)
	}

	return r
}
