package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type QueueHandler struct {
	queue service.QueueService
}

func NewQueueHandler(queue service.QueueService) *QueueHandler {
	return &QueueHandler{queue: queue}
}

// Join godoc
//
//	@Summary		Join the virtual queue for a showtime
//	@Description	Only needed when GET /shows/:id/seats or a hold attempt reports a queue is active for that showtime.
//	@Tags			queue
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.QueueJoinRequest	true	"Showtime"
//	@Success		201		{object}	response.Body{data=dto.QueueJoinResponse}
//	@Router			/queue/join [post]
func (h *QueueHandler) Join(c *gin.Context) {
	var req dto.QueueJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	position, err := h.queue.Join(c.Request.Context(), req.ShowtimeID, middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.QueueJoinResponse{Position: position})
}

// Status godoc
//
//	@Summary		Poll queue status; admitted=true means Hold will now accept this user for this showtime
//	@Tags			queue
//	@Produce		json
//	@Security		BearerAuth
//	@Param			showtime_id	query		string	true	"Showtime ID"
//	@Success		200			{object}	response.Body{data=dto.QueueStatusResponse}
//	@Failure		404			{object}	response.Body
//	@Router			/queue/status [get]
func (h *QueueHandler) Status(c *gin.Context) {
	res, err := h.queue.Status(c.Request.Context(), c.Query("showtime_id"), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
