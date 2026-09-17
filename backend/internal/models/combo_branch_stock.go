package models

import "time"

// ComboBranchStock: no row for a (branch, combo) pair means unlimited stock
// (Phần 2.2). Reserved at hold time, released on cancel/expire/refund.
type ComboBranchStock struct {
	BranchID      string    `gorm:"type:uuid;primaryKey" json:"branch_id"`
	ComboID       string    `gorm:"type:uuid;primaryKey" json:"combo_id"`
	StockQuantity int       `gorm:"not null" json:"stock_quantity"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ComboBranchStock) TableName() string { return "combo_branch_stock" }
