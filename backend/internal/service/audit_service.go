package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type AuditService interface {
	List(ctx context.Context, query dto.AuditLogListQuery) ([]dto.AuditLogResponse, int64, error)
}

type auditService struct {
	repo repository.AuditRepository
}

func NewAuditService(repo repository.AuditRepository) AuditService {
	return &auditService{repo: repo}
}

func (s *auditService) List(ctx context.Context, query dto.AuditLogListQuery) ([]dto.AuditLogResponse, int64, error) {
	if query.From != "" {
		if _, err := time.Parse(dto.DateLayout, query.From); err != nil {
			return nil, 0, apperrors.Validation("from must follow format YYYY-MM-DD")
		}
	}
	if query.To != "" {
		if _, err := time.Parse(dto.DateLayout, query.To); err != nil {
			return nil, 0, apperrors.Validation("to must follow format YYYY-MM-DD")
		}
	}
	rows, total, err := s.repo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.AuditLogResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.NewAuditLogResponse(row))
	}
	return out, total, nil
}

