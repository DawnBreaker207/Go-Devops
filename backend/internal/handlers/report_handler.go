package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// ReportHandler serves admin reports (money: admin only).
type ReportHandler struct {
	reports service.ReportService
}

func NewReportHandler(reports service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

// Daily godoc
//
//	@Summary		Revenue per closed day (admin)
//	@Description	Reads daily_aggregates written by closeDay: confirmed bookings by payment time. Default range: last 7 days. Staff and customers get 403.
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from	query		string	false	"YYYY-MM-DD"
//	@Param			to		query		string	false	"YYYY-MM-DD (default today)"
//	@Success		200		{object}	response.Body{data=dto.DailyReportResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Router			/admin/reports/daily [get]
func (h *ReportHandler) Daily(c *gin.Context) {
	report, err := h.reports.DailyReport(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, report)
}
