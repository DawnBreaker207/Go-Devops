package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

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

// Overview godoc
//
//	@Summary		Admin dashboard: today (live), the last 7 days, upcoming showtimes and operational alerts, in one call
//	@Description	Today's revenue/occupancy is computed live, not waiting on the closeDay job. Alerts surface stuck refunds, recent failed batch jobs and confirmed orders whose ticket email exhausted every retry.
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=dto.AdminOverviewResponse}
//	@Failure		403	{object}	response.Body
//	@Router			/admin/overview [get]
func (h *ReportHandler) Overview(c *gin.Context) {
	res, err := h.reports.AdminOverview(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
