package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type AuditHandler struct {
	audit service.AuditService
}

func NewAuditHandler(audit service.AuditService) *AuditHandler {
	return &AuditHandler{audit: audit}
}

// List godoc
//
//	@Summary		List audit log entries (admin)
//	@Description	Every audited event, newest first. booking_id ties together the
//	@Description	whole lifecycle of one order (hold/pay/webhook/refund/redeem)
//	@Description	even though they touch different resource_type/resource_id.
//	@Tags			admin-audit
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query	int		false	"Page"
//	@Param			page_size	query	int		false	"Page size"
//	@Param			action		query	string	false	"Exact action, e.g. orders.pay"
//	@Param			resource_type	query	string	false	"e.g. booking, payment, ticket, movie, hall"
//	@Param			resource_id	query	string	false	"Exact resource id"
//	@Param			booking_id	query	string	false	"Every event of one order's lifecycle"
//	@Param			actor_id	query	string	false	"Who did it"
//	@Param			outcome		query	string	false	"success | failure"
//	@Param			from		query	string	false	"YYYY-MM-DD, inclusive"
//	@Param			to			query	string	false	"YYYY-MM-DD, inclusive"
//	@Success		200	{object}	response.Body{data=response.Paged{items=[]dto.AuditLogResponse}}
//	@Failure		400	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Router			/admin/audit-logs [get]
func (h *AuditHandler) List(c *gin.Context) {
	var query dto.AuditLogListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	rows, total, err := h.audit.List(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, query.Page, query.PageSize, total)
}
