package handlers

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/response"
)

// PaymentHandler exposes the provider-agnostic payment endpoints. The mock and
// real gateways are served by the same three routes.
type PaymentHandler struct {
	providers         *payment.Registry
	bookings          service.BookingService
	returnRedirectURL string
}

// NewPaymentHandler builds the handler; an empty returnRedirectURL answers the
// return route with JSON instead of redirecting to the frontend.
func NewPaymentHandler(providers *payment.Registry, bookings service.BookingService, returnRedirectURL string) *PaymentHandler {
	return &PaymentHandler{providers: providers, bookings: bookings, returnRedirectURL: returnRedirectURL}
}

// Providers godoc
//
//	@Summary		List the payment providers a customer can choose
//	@Tags			payments
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Body{data=[]dto.PaymentProviderResponse}
//	@Failure		401	{object}	response.Body
//	@Router			/payments/providers [get]
func (h *PaymentHandler) Providers(c *gin.Context) {
	items := make([]dto.PaymentProviderResponse, 0)
	for _, p := range h.providers.List() {
		items = append(items, dto.PaymentProviderResponse{
			Name:        p.Name(),
			DisplayName: p.DisplayName(),
			Default:     p.Name() == h.providers.Default(),
		})
	}
	response.OK(c, items)
}

// Notify godoc
//
//	@Summary		Provider IPN callback
//	@Description	Outside JWT: the provider signature is the credential. GET or POST depending on the gateway; the answer follows the gateway's own contract.
//	@Tags			payments
//	@Param			provider	path	string	true	"Provider name, e.g. mock"
//	@Success		200
//	@Failure		401
//	@Failure		404
//	@Router			/payments/{provider}/ipn [post]
func (h *PaymentHandler) Notify(c *gin.Context) {
	provider, ok := h.providers.Get(c.Param("provider"))
	if !ok {
		response.Error(c, apperrors.NotFound("unknown payment provider"))
		return
	}
	// WHO/IP for the audit rows the service writes, accepted or rejected.
	ctx := audit.Stash(c.Request.Context(), audit.FromGin(c, audit.Record{ActorRole: "provider:" + provider.Name()}))
	ack := h.bookings.HandleNotification(ctx, provider, c.Request)
	provider.AckNotification(c.Writer, ack)
}

// Return godoc
//
//	@Summary		Browser return from a provider checkout
//	@Description	Verifies the redirect, reconciles with the provider, then redirects to payment.return_redirect_url (or answers JSON when unset).
//	@Tags			payments
//	@Produce		json
//	@Param			provider	path	string	true	"Provider name, e.g. mock"
//	@Success		200	{object}	response.Body{data=dto.PaymentReturnResponse}
//	@Success		303
//	@Failure		401	{object}	response.Body
//	@Failure		404	{object}	response.Body
//	@Router			/payments/{provider}/return [get]
func (h *PaymentHandler) Return(c *gin.Context) {
	provider, ok := h.providers.Get(c.Param("provider"))
	if !ok {
		response.Error(c, apperrors.NotFound("unknown payment provider"))
		return
	}
	res, err := h.bookings.HandleReturn(c.Request.Context(), provider, c.Request)
	if err != nil {
		response.Error(c, err)
		return
	}
	if h.returnRedirectURL == "" {
		response.OK(c, res)
		return
	}
	target, err := url.Parse(h.returnRedirectURL)
	if err != nil {
		response.Error(c, apperrors.Internal("invalid payment return redirect url").Wrap(err))
		return
	}
	q := target.Query()
	q.Set("booking_id", res.BookingID)
	q.Set("status", res.BookingStatus)
	q.Set("payment_status", res.PaymentStatus)
	target.RawQuery = q.Encode()
	c.Redirect(http.StatusSeeOther, target.String())
}
