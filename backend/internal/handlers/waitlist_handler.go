package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type WaitlistHandler struct {
	waitlist service.WaitlistService
}

func NewWaitlistHandler(waitlist service.WaitlistService) *WaitlistHandler {
	return &WaitlistHandler{waitlist: waitlist}
}

// Join godoc
//
//	@Summary		Wait for a seat on a sold-out showtime
//	@Description	Refused while the showtime still has an available seat. When a held seat is later released, the oldest waiting entry gets it held for them (same TTL as a normal hold) — check GET /waitlist/me or GET /orders.
//	@Tags			waitlist
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.WaitlistJoinRequest	true	"Showtime"
//	@Success		201		{object}	response.Body{data=dto.WaitlistEntryResponse}
//	@Failure		409		{object}	response.Body
//	@Router			/waitlist [post]
func (h *WaitlistHandler) Join(c *gin.Context) {
	var req dto.WaitlistJoinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	entry, err := h.waitlist.Join(c.Request.Context(), middleware.CurrentUserID(c), req.ShowtimeID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, entry)
}

// Cancel godoc
//
//	@Summary		Leave a waitlist
//	@Tags			waitlist
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Waitlist entry ID"
//	@Success		200	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/waitlist/{id} [delete]
func (h *WaitlistHandler) Cancel(c *gin.Context) {
	if err := h.waitlist.Cancel(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id")); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "canceled")
}

// My godoc
//
//	@Summary		My waitlist entries
//	@Tags			waitlist
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.WaitlistEntryResponse}
//	@Router			/waitlist/me [get]
func (h *WaitlistHandler) My(c *gin.Context) {
	rows, err := h.waitlist.MyEntries(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}
