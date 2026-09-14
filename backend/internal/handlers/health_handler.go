package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/database"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type HealthHandler struct {
	db      *gorm.DB
	appName string
}

func NewHealthHandler(db *gorm.DB, appName string) *HealthHandler {
	return &HealthHandler{db: db, appName: appName}
}

type HealthStatus struct {
	Status   string `json:"status" example:"ok"`
	Service  string `json:"service" example:"BackEnd-CP"`
	Database string `json:"database" example:"up"`
}

// Check godoc
//
//	@Summary		Health check
//	@Description	Service and database status
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	response.Body{data=handlers.HealthStatus}
//	@Failure		503	{object}	response.Body
//	@Router			/health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	status := HealthStatus{Status: "ok", Service: h.appName, Database: "up"}

	if err := database.Healthy(h.db, 2*time.Second); err != nil {
		status.Status = "degraded"
		status.Database = "down"
		c.JSON(http.StatusServiceUnavailable, response.Body{
			Code:    http.StatusServiceUnavailable,
			Message: "database unavailable",
			Data:    status,
		})
		return
	}

	response.OK(c, status)
}

// Healthz godoc
//
//	@Summary		Infrastructure probe
//	@Description	200 when the DB pings, 503 otherwise; no complex payload.
//	@Tags			health
//	@Produce		json
//	@Success		200	{object}	response.Body
//	@Failure		503	{object}	response.Body
//	@Router			/healthz [get]
func (h *HealthHandler) Healthz(c *gin.Context) {
	if err := database.Healthy(h.db, 2*time.Second); err != nil {
		c.JSON(http.StatusServiceUnavailable, response.Body{
			Code:    http.StatusServiceUnavailable,
			Message: "unavailable",
		})
		return
	}
	response.OK(c, nil)
}
