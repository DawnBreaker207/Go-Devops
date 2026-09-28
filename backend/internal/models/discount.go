package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// New values need the Go const, the `oneof=` DTO tag and the SQL CHECK.
const (
	DiscountPercent = "percent"
	DiscountAmount = "amount"
)

// Display order; the frontend hard-codes it too.
var AllDiscountKinds = []string{DiscountPercent, DiscountAmount}

// Never touches bookings.total_amount (see Booking.Payable).
// Soft delete keeps past bookings' discount_code_id valid; Active is the on/off switch.
type DiscountCode struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	// Stored and matched UPPERCASE; the service uppercases on the way in.
	Code        string `gorm:"type:varchar(32);not null" json:"code"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Kind        string `gorm:"type:varchar(16);not null" json:"kind"`
	Value       int64  `gorm:"not null" json:"value"`
	// Caps percent codes; meaningless (rejected) for DiscountAmount.
	MaxDiscount *int64 `json:"max_discount,omitempty"`
	// Subtotal required; 0 = no minimum.
	MinOrder int64 `gorm:"not null;default:0" json:"min_order"`
	// Nil on either side means open-ended in that direction.
	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
	MaxUses   *int `json:"max_uses,omitempty"`
	UsedCount int  `gorm:"not null;default:0" json:"used_count"`
	// CampaignID nil = standalone; a linked code also needs its campaign running.
	CampaignID *string `gorm:"type:uuid" json:"campaign_id,omitempty"`
	// No `default:` tag: GORM omits zero-valued fields with a default, silently re-enabling active=false.
	Active    bool           `gorm:"not null" json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DiscountCode) TableName() string { return "discount_codes" }

func (d *DiscountCode) BeforeCreate(*gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	return nil
}

// DiscountFor answers the arithmetic only; validity is the service's job.
func (d *DiscountCode) DiscountFor(subtotal int64) int64 {
	if subtotal <= 0 {
		return 0
	}
	var off int64
	switch d.Kind {
	case DiscountPercent:
		// Integer maths on whole VND: this truncates, so the house rounds in the
		// customer's disfavour by at most 1 VND rather than giving money away.
		off = subtotal * d.Value / 100
		if d.MaxDiscount != nil && off > *d.MaxDiscount {
			off = *d.MaxDiscount
		}
	case DiscountAmount:
		off = d.Value
	default:
		return 0
	}
	if off > subtotal {
		off = subtotal
	}
	if off < 0 {
		return 0
	}
	return off
}
