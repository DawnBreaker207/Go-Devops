package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Combo is a concession/combo product sold independently of a booking (F5,
// combo/concession upsell). Soft delete keeps old combo orders readable after
// a product is retired; Active is the separate on/off-sale flag.
type Combo struct {
	ID          string `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Price       int64  `gorm:"not null" json:"price"`
	ImageURL    string `gorm:"type:varchar(512)" json:"image_url,omitempty"`
	// No `default:true` here, unlike User.Active and Hall.Active, and that is
	// deliberate: GORM OMITS a zero-valued field from an INSERT when the column
	// carries a default, so a product created as active=false would silently come
	// back on sale. This is the only table whose create endpoint accepts the flag,
	// which is why only this one needs the tag dropped. The SQL column keeps its
	// own DEFAULT TRUE for raw inserts such as the seed file, and there is no
	// AutoMigrate in this repo, so removing the tag changes no schema.
	Active    bool           `gorm:"not null" json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName is "concession_items": "combos" is taken by another in-progress feature.
func (Combo) TableName() string { return "concession_items" }

func (c *Combo) BeforeCreate(*gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	return nil
}

const (
	ComboOrderPending   = "pending"
	ComboOrderConfirmed = "confirmed"
	ComboOrderCancelled = "cancelled"
)

// ComboOrder is a combo/concession purchase. It is intentionally independent of
// the ticket booking flow: BookingID is an optional correlation only (e.g. "pick
// up with your popcorn at the counter"), never a foreign key a booking write
// depends on — a combo order failing must never roll back or block a booking.
type ComboOrder struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    string    `gorm:"type:uuid;not null;index" json:"user_id"`
	BookingID *string   `gorm:"type:uuid" json:"booking_id,omitempty"`
	Status    string    `gorm:"type:varchar(16);not null;default:confirmed" json:"status"`
	Total     int64     `gorm:"not null;default:0" json:"total"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ComboOrder) TableName() string { return "combo_orders" }

func (o *ComboOrder) BeforeCreate(*gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	return nil
}

// ComboOrderItem snapshots the combo's name and price at purchase time, the
// same convention BookingSeat uses for seat type/price: a later rename or
// price change of the combo must never alter a past order's receipt.
type ComboOrderItem struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	ComboOrderID string    `gorm:"type:uuid;not null;index" json:"combo_order_id"`
	ComboID      string    `gorm:"type:uuid;not null" json:"combo_id"`
	ComboName    string    `gorm:"type:varchar(255);not null" json:"combo_name"`
	Quantity     int       `gorm:"not null" json:"quantity"`
	UnitPrice    int64     `gorm:"not null" json:"unit_price"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ComboOrderItem) TableName() string { return "combo_order_items" }

func (i *ComboOrderItem) BeforeCreate(*gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.NewString()
	}
	return nil
}
