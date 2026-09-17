package repository

import (
	"context"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type LedgerRepository struct {
	db *gorm.DB
}

func NewLedgerRepository(db *gorm.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

func (r *LedgerRepository) Write(tx *gorm.DB, entry *models.LedgerEntry) error {
	return tx.Create(entry).Error
}

// Summary is the whole point of unifying every money movement into one
// table (Phần 2.5): a revenue breakdown is one GROUP BY, not a UNION ALL.
type LedgerSummaryRow struct {
	Type  string
	Total int64
	Count int64
}

func (r *LedgerRepository) Summary(ctx context.Context, from, to string) ([]LedgerSummaryRow, error) {
	var rows []LedgerSummaryRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT type, SUM(amount) AS total, COUNT(*) AS count
		FROM ledger_entries
		WHERE created_at >= ? AND created_at < ?
		GROUP BY type ORDER BY type`, from, to).Scan(&rows).Error
	return rows, err
}
