package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
)

type AuditRepository interface {
	List(ctx context.Context, query dto.AuditLogListQuery) ([]models.AuditLog, int64, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) List(ctx context.Context, query dto.AuditLogListQuery) ([]models.AuditLog, int64, error) {
	tx := r.db.WithContext(ctx).Model(&models.AuditLog{})
	if query.Action != "" {
		tx = tx.Where("action = ?", query.Action)
	}
	if query.ResourceType != "" {
		tx = tx.Where("resource_type = ?", query.ResourceType)
	}
	if query.ResourceID != "" {
		tx = tx.Where("resource_id = ?", query.ResourceID)
	}
	if query.BookingID != "" {
		tx = tx.Where("booking_id = ?", query.BookingID)
	}
	if query.ActorID != "" {
		tx = tx.Where("actor_id = ?", query.ActorID)
	}
	if query.Outcome != "" {
		tx = tx.Where("outcome = ?", query.Outcome)
	}
	if query.From != "" {
		from, err := time.Parse(dto.DateLayout, query.From)
		if err != nil {
			return nil, 0, fmt.Errorf("parse from: %w", err)
		}
		tx = tx.Where("created_at >= ?", from)
	}
	if query.To != "" {
		to, err := time.Parse(dto.DateLayout, query.To)
		if err != nil {
			return nil, 0, fmt.Errorf("parse to: %w", err)
		}
		tx = tx.Where("created_at < ?", to.AddDate(0, 0, 1))
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}
	rows := make([]models.AuditLog, 0, query.PageSize)
	if err := tx.Order("created_at DESC").Limit(query.PageSize).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return rows, total, nil
}

