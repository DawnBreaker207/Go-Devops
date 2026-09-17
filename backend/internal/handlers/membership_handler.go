package handlers

import (
	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/middleware"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

type MembershipHandler struct {
	membershipService service.MembershipService
}

func NewMembershipHandler(membershipService service.MembershipService) *MembershipHandler {
	return &MembershipHandler{membershipService: membershipService}
}

// ListTiers godoc
//
//	@Summary		List active membership tiers (public)
//	@Tags			membership
//	@Produce		json
//	@Success		200	{object}	response.Body{data=[]dto.MembershipTierResponse}
//	@Router			/membership-tiers [get]
func (h *MembershipHandler) ListTiers(c *gin.Context) {
	tiers, err := h.membershipService.ListTiers(c.Request.Context(), true)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tiers)
}

// ListTiersAdmin godoc
//
//	@Summary		List all membership tiers, including inactive (admin)
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.MembershipTierResponse}
//	@Router			/admin/membership-tiers [get]
func (h *MembershipHandler) ListTiersAdmin(c *gin.Context) {
	tiers, err := h.membershipService.ListTiers(c.Request.Context(), false)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tiers)
}

// CreateTier godoc
//
//	@Summary		Create a membership tier
//	@Tags			membership
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.MembershipTierRequest	true	"Tier"
//	@Success		201		{object}	response.Body{data=dto.MembershipTierResponse}
//	@Failure		400		{object}	response.Body
//	@Router			/admin/membership-tiers [post]
func (h *MembershipHandler) CreateTier(c *gin.Context) {
	var req dto.MembershipTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	tier, err := h.membershipService.CreateTier(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, tier)
}

// UpdateTier godoc
//
//	@Summary		Update a membership tier
//	@Tags			membership
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string							true	"Tier ID"
//	@Param			payload	body		dto.UpdateMembershipTierRequest	true	"Fields to change"
//	@Success		200		{object}	response.Body{data=dto.MembershipTierResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		404		{object}	response.Body
//	@Router			/admin/membership-tiers/{id} [put]
func (h *MembershipHandler) UpdateTier(c *gin.Context) {
	var req dto.UpdateMembershipTierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	tier, err := h.membershipService.UpdateTier(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, tier)
}

// Purchase godoc
//
//	@Summary		Buy or renew a membership
//	@Description	Buying while already holding an active membership of the same tier renews it (extends expires_at); a different tier while active is refused.
//	@Tags			membership
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			payload	body		dto.PurchaseMembershipRequest	true	"Purchase"
//	@Success		200		{object}	response.Body{data=dto.UserMembershipResponse}
//	@Failure		400		{object}	response.Body
//	@Failure		409		{object}	response.Body
//	@Router			/memberships/purchase [post]
func (h *MembershipHandler) Purchase(c *gin.Context) {
	var req dto.PurchaseMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	m, err := h.membershipService.Purchase(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, m)
}

// My godoc
//
//	@Summary		List my memberships
//	@Tags			membership
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.UserMembershipResponse}
//	@Router			/memberships/me [get]
func (h *MembershipHandler) My(c *gin.Context) {
	rows, err := h.membershipService.MyMemberships(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, rows)
}
