package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type LedgerHandler struct {
	ledger service.LedgerService
}

func NewLedgerHandler(ledger service.LedgerService) *LedgerHandler {
	return &LedgerHandler{ledger: ledger}
}

// Summary godoc
//
//	@Summary		Revenue/points breakdown by ledger entry type
//	@Description	One row per type (payment_captured, payment_refunded, membership_purchase, reward_redeemed); default range is the last 7 days.
//	@Tags			reports
//	@Produce		json
//	@Security		BearerAuth
//	@Param			from	query		string	false	"YYYY-MM-DD"
//	@Param			to		query		string	false	"YYYY-MM-DD"
//	@Success		200		{object}	response.Body{data=[]dto.LedgerSummaryResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/ledger/summary [get]
func (h *LedgerHandler) Summary(c *gin.Context) {
	rows, err := h.ledger.Summary(c.Request.Context(), c.Query("from"), c.Query("to"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}
