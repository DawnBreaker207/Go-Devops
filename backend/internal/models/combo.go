package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Concession product independent of bookings; soft delete keeps old orders readable.
type Combo struct {
	ID          string `gorm:"type:uuid;primaryKey" json:"id"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	Price       int64  `gorm:"not null" json:"price"`
	ImageURL    string `gorm:"type:varchar(512)" json:"image_url,omitempty"`
	// No `default:true`: GORM omits zero-valued fields with a default, silently re-enabling active=false.
	Active    bool           `gorm:"not null" json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// "combos" is taken by another feature, hence "concession_items".
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
	ComboOrderCollected = "collected"
)

const (
	SoldChannelOnline  = "online"
	SoldChannelCounter = "counter"
)

const (
	PayMethodCash = "cash"
	PayMethodPOS  = "pos"
)

// BookingID is optional correlation; NULL UserID is a walk-in counter sale.
type ComboOrder struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID       *string   `gorm:"type:uuid;index" json:"user_id,omitempty"`
	BookingID    *string   `gorm:"type:uuid" json:"booking_id,omitempty"`
	Status       string    `gorm:"type:varchar(16);not null;default:confirmed" json:"status"`
	Total        int64     `gorm:"not null;default:0" json:"total"`
	SoldChannel  string    `gorm:"type:varchar(16);not null;default:online" json:"sold_channel"`
	PayMethod    *string   `gorm:"type:varchar(16)" json:"pay_method,omitempty"`
	CustomerName *string   `gorm:"type:varchar(255)" json:"customer_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ComboOrder) TableName() string { return "combo_orders" }

func (o *ComboOrder) BeforeCreate(*gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	return nil
}

// Snapshots name/price at purchase time.
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
