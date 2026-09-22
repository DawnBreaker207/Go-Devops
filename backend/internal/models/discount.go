package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Discount kinds. A new value needs the Go const, the `oneof=` binding tag on
// every DTO that accepts it, and the SQL CHECK in a NEW migration.
const (
	// DiscountPercent takes Value percent off, capped by MaxDiscount when set.
	DiscountPercent = "percent"
	// DiscountAmount takes a flat Value VND off.
	DiscountAmount = "amount"
)

// AllDiscountKinds is the display order; no endpoint returns this list, so the
// frontend hard-codes it too (src/types/discount.ts).
var AllDiscountKinds = []string{DiscountPercent, DiscountAmount}

// DiscountCode is a code a customer applies to a PENDING order before paying.
// It never touches bookings.total_amount, which stays the seat subtotal: see
// Booking.Payable and migration 000010 for why.
//
// Soft delete keeps past bookings' discount_code_id valid after a code is
// retired; Active is the separate on/off switch an operator flips day to day.
type DiscountCode struct {
	ID string `gorm:"type:uuid;primaryKey" json:"id"`
	// Stored and matched UPPERCASE; the service uppercases on the way in.
	Code        string `gorm:"type:varchar(32);not null" json:"code"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Kind        string `gorm:"type:varchar(16);not null" json:"kind"`
	Value       int64  `gorm:"not null" json:"value"`
	// MaxDiscount caps a percentage code; nil means uncapped. Meaningless for
	// DiscountAmount and rejected there by the service.
	MaxDiscount *int64 `json:"max_discount,omitempty"`
	// MinOrder is the subtotal required before the code applies. 0 = no minimum.
	MinOrder int64 `gorm:"not null;default:0" json:"min_order"`
	// Nil on either side means open-ended in that direction.
	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
	// MaxUses nil means unlimited.
	MaxUses   *int `json:"max_uses,omitempty"`
	UsedCount int  `gorm:"not null;default:0" json:"used_count"`
	// No `default:` tag on purpose - GORM omits a zero-valued field from an
	// INSERT when the column has a default, so a code created as active=false
	// would silently come back on. Same trap as models.Combo.Active.
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

// DiscountFor computes what this code takes off `subtotal`, in whole VND, and
// never returns more than the subtotal. It answers the arithmetic only —
// validity (active, window, uses, minimum order) is the service's job.
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
