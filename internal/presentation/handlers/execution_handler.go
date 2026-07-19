package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gabrielAnFran/pos-production-service/internal/application/usecases"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/entities"
	"github.com/gabrielAnFran/pos-production-service/internal/domain/repositories"
	"github.com/gabrielAnFran/pos-production-service/internal/presentation/dto"
	"github.com/gabrielAnFran/pos-production-service/internal/presentation/middleware"
)

type ExecutionHandler struct {
	repo    repositories.ExecutionRepository
	updater *usecases.UpdateExecutionStatusUseCase
}

func NewExecutionHandler(repo repositories.ExecutionRepository, updater *usecases.UpdateExecutionStatusUseCase) *ExecutionHandler {
	return &ExecutionHandler{repo: repo, updater: updater}
}

func (h *ExecutionHandler) Get(c *gin.Context) {
	osID := c.Param("os_id")
	exec, err := h.repo.FindByOSID(c.Request.Context(), osID)
	if err != nil {
		if errors.Is(err, repositories.ErrNotFound) {
			dto.WriteProblem(c, http.StatusNotFound, "execution not found", "no execution found for os_id "+osID)
			return
		}
		dto.WriteProblem(c, http.StatusInternalServerError, "internal error", err.Error())
		return
	}
	c.JSON(http.StatusOK, dto.FromEntity(exec))
}

func (h *ExecutionHandler) UpdateStatus(c *gin.Context) {
	osID := c.Param("os_id")

	var req dto.UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.WriteProblem(c, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}

	correlationID := middleware.GetCorrelationID(c)
	exec, err := h.updater.Handle(c.Request.Context(), osID, req.Status, req.Notes, correlationID)
	if err != nil {
		switch {
		case errors.Is(err, repositories.ErrNotFound):
			dto.WriteProblem(c, http.StatusNotFound, "execution not found", "no execution found for os_id "+osID)
		case errors.Is(err, entities.ErrInvalidTransition):
			dto.WriteProblem(c, http.StatusBadRequest, "invalid status transition", err.Error())
		default:
			dto.WriteProblem(c, http.StatusInternalServerError, "internal error", err.Error())
		}
		return
	}
	c.JSON(http.StatusOK, dto.FromEntity(exec))
}
