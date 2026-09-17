package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

type LedgerService interface {
	Summary(ctx context.Context, from, to string) ([]dto.LedgerSummaryResponse, error)
}

type ledgerService struct {
	repo *repository.LedgerRepository
}

func NewLedgerService(repo *repository.LedgerRepository) LedgerService {
	return &ledgerService{repo: repo}
}

func (s *ledgerService) Summary(ctx context.Context, from, to string) ([]dto.LedgerSummaryResponse, error) {
	if from == "" {
		from = time.Now().AddDate(0, 0, -7).Format(dto.DateLayout)
	}
	if to == "" {
		to = time.Now().Format(dto.DateLayout)
	}
	fromT, err := time.Parse(dto.DateLayout, from)
	if err != nil {
		return nil, apperrors.Validation("from must follow format YYYY-MM-DD")
	}
	toT, err := time.Parse(dto.DateLayout, to)
	if err != nil {
		return nil, apperrors.Validation("to must follow format YYYY-MM-DD")
	}
	rows, err := s.repo.Summary(ctx, fromT.Format(time.RFC3339), toT.AddDate(0, 0, 1).Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	out := make([]dto.LedgerSummaryResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.LedgerSummaryResponse{Type: r.Type, Total: r.Total, Count: r.Count})
	}
	return out, nil
}
