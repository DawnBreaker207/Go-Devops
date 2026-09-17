package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type LoyaltyHandler struct {
	loyalty service.LoyaltyService
}

func NewLoyaltyHandler(loyalty service.LoyaltyService) *LoyaltyHandler {
	return &LoyaltyHandler{loyalty: loyalty}
}

// Balance godoc
//
//	@Summary		My points balance
//	@Tags			loyalty
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=dto.PointsBalanceResponse}
//	@Router			/loyalty/points [get]
func (h *LoyaltyHandler) Balance(c *gin.Context) {
	balance, err := h.loyalty.Balance(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, balance)
}

// Transactions godoc
//
//	@Summary		My points history
//	@Tags			loyalty
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Current page"	default(1)
//	@Param			page_size	query		int	false	"Records per page"	default(10)
//	@Success		200			{object}	response.Body{data=response.Paged{items=[]dto.PointTransactionResponse}}
//	@Router			/loyalty/points/transactions [get]
func (h *LoyaltyHandler) Transactions(c *gin.Context) {
	var query dto.PageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.Error(c, err)
		return
	}
	query.Normalize()
	rows, total, err := h.loyalty.Transactions(c.Request.Context(), middleware.CurrentUserID(c), query)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, rows, query.Page, query.PageSize, total)
}

// ListRewards godoc
//
//	@Summary		List active rewards (public/customer)
//	@Tags			loyalty
//	@Produce		json
//	@Success		200	{object}	response.Body{data=[]dto.RewardResponse}
//	@Router			/rewards [get]
func (h *LoyaltyHandler) ListRewards(c *gin.Context) {
	rows, err := h.loyalty.ListRewards(c.Request.Context(), true)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// ListRewardsAdmin godoc
//
//	@Summary		List all rewards, including inactive (admin)
//	@Tags			loyalty
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.RewardResponse}
//	@Router			/admin/rewards [get]
func (h *LoyaltyHandler) ListRewardsAdmin(c *gin.Context) {
	rows, err := h.loyalty.ListRewards(c.Request.Context(), false)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}

// CreateReward godoc
//
//	@Summary		Create a reward
//	@Tags			loyalty
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.RewardRequest	true	"Reward"
//	@Success		201		{object}	response.Body{data=dto.RewardResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/rewards [post]
func (h *LoyaltyHandler) CreateReward(c *gin.Context) {
	var req dto.RewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	reward, err := h.loyalty.CreateReward(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, reward)
}

// UpdateReward godoc
//
//	@Summary		Update a reward
//	@Tags			loyalty
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string					true	"Reward ID"
//	@Param			payload	body		dto.UpdateRewardRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.RewardResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/rewards/{id} [put]
func (h *LoyaltyHandler) UpdateReward(c *gin.Context) {
	var req dto.UpdateRewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	reward, err := h.loyalty.UpdateReward(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, reward)
}

// Redeem godoc
//
//	@Summary		Redeem points for a reward
//	@Description	Not refundable/cancelable. A gift is picked up at the counter by its code; a voucher reward is delivered at once.
//	@Tags			loyalty
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Reward ID"
//	@Success		201	{object}	response.Body{data=dto.RewardRedemptionResponse}
//	@Failure		400	{object}	response.Body
//	@Failure		409	{object}	response.Body
//	@Router			/rewards/{id}/redeem [post]
func (h *LoyaltyHandler) Redeem(c *gin.Context) {
	red, err := h.loyalty.Redeem(c.Request.Context(), middleware.CurrentUserID(c), c.Param("id"))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, red)
}

// MarkDelivered godoc
//
//	@Summary		Mark a gift redemption delivered (staff, counter pickup)
//	@Tags			loyalty
//	@Produce		json
//	@Security		BearerAuth
//	@Param			code	path	string	true	"Redemption code"
//	@Success		200		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/staff/rewards/{code}/deliver [post]
func (h *LoyaltyHandler) MarkDelivered(c *gin.Context) {
	if err := h.loyalty.MarkDelivered(c.Request.Context(), c.Param("code")); err != nil {
		response.Error(c, err)
		return
	}
	response.NoContentOK(c, "delivered")
}
