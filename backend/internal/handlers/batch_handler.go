package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type BatchHandler struct {
	manager *batch.Manager
	repo    repository.BatchJobRepository
	db      *gorm.DB
}

func NewBatchHandler(manager *batch.Manager, repo repository.BatchJobRepository, db *gorm.DB) *BatchHandler {
	return &BatchHandler{manager: manager, repo: repo, db: db}
}

// List godoc
//
//	@Summary		Batch job run history
//	@Tags			batch
//	@Produce		json
//	@Param			page		query	int	false	"Page"
//	@Param			page_size	query	int	false	"Page size"
//	@Param			search		query	string	false	"Filter by job_name"
//	@Success		200			{object}	response.Body{data=response.Paged}
//	@Failure		401			{object}	response.Body
//	@Failure		403			{object}	response.Body
//	@Router			/admin/batch/jobs [get]
func (h *BatchHandler) List(c *gin.Context) {
	var q dto.PageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, err)
		return
	}
	q.Normalize()

	jobs, total, err := h.repo.List(c.Request.Context(), q)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, jobs, q.Page, q.PageSize, total)
}

// Run godoc
//
//	@Summary		Start a job now, in the background
//	@Description	Answers 202 once the run is registered; follow its status in /admin/batch/jobs.
//	@Tags			batch
//	@Accept			json
//	@Produce		json
//	@Param			name	path	string	true	"Job name (e.g. closeDay, sweepExpiredHolds)"
//	@Success		202		{object}	response.Body
//	@Failure		401		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/admin/batch/jobs/{name}/run [post]
func (h *BatchHandler) Run(c *gin.Context) {
	name := c.Param("name")
	// Not run in this request: a long job would outlast the write timeout and
	// a disconnected client would cancel it.
	runID, err := h.manager.Trigger(name, models.TriggerManual)
	if err != nil {
		response.Error(c, err)
		return
	}
	if rec, ok := audit.FromContext(c.Request.Context()); ok {
		rec.ResourceID = name
		rec.After = map[string]any{"triggered_by": models.TriggerManual, "run_id": runID}
		if err := audit.In(c.Request.Context(), h.db, rec); err != nil {
			logger.L().Warn("batch run audit row not written", logger.Err(err))
		}
	}
	c.JSON(http.StatusAccepted, response.Body{Code: response.CodeSuccess, Message: "accepted", Data: gin.H{
		"job": name, "run_id": runID, "status": models.BatchRunning, "triggered_by": models.TriggerManual,
	}})
}
