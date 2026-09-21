package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type ReportHandler struct {
	reports service.ReportService
}

func NewReportHandler(reports service.ReportService) *ReportHandler {
	return &ReportHandler{reports: reports}
}

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
	// Typed explicitly so this file imports internal/dto: swag resolves the `dto.` prefix
	// in the annotations from the file's own imports, and without it `make swag` aborts.
	var report *dto.DailyReportResponse
	report, err := h.reports.DailyReport(c.Request.Context(), c.Query("from"), c.Query("to")) //nolint:staticcheck // keeps the dto import load-bearing
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, report)
}

//	@Summary		Revenue analytics breakdown (admin)
//	@Description	Live aggregation over paid money in [from, to]: daily line, top movies/halls and the payment-method split. Same money rule as closeDay (confirmed bookings by payment time). Default range: last 7 days. Staff and customers get 403.
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from	query		string	false	"YYYY-MM-DD"
//	@Param			to		query		string	false	"YYYY-MM-DD (default today)"
//	@Success		200		{object}	response.Body{data=dto.BreakdownResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		403		{object}	response.Body
//	@Router			/admin/reports/breakdown [get]
func (h *ReportHandler) Breakdown(c *gin.Context) {
	// Typed explicitly so this file imports internal/dto: swag resolves the `dto.` prefix
	// in the annotations from the file's own imports, and without it `make swag` aborts.
	var report *dto.BreakdownResponse
	report, err := h.reports.Breakdown(c.Request.Context(), c.Query("from"), c.Query("to")) //nolint:staticcheck // keeps the dto import load-bearing
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, report)
}

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

//	@Summary		Admin dashboard headline counts
//	@Description	Four totals for the dashboard tiles: movies, showtimes, bookings and users. Soft-deleted rows are excluded and bookings counts confirmed orders only, so pending holds and expired bookings never inflate the tile. Locked accounts are still counted. Staff and customers get 403.
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=dto.AdminStatsResponse}
//	@Failure		401	{object}	response.Body
//	@Failure		403	{object}	response.Body
//	@Router			/admin/stats [get]
func (h *ReportHandler) Stats(c *gin.Context) {
	res, err := h.reports.AdminStats(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, res)
}
