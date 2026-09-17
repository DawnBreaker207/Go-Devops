package service

import (
	"context"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// maxActiveWaitlistEntries: how many showtimes a customer may wait on at
// once (Product Backlog "Giới hạn số suất/user được đăng ký chờ cùng lúc").
const maxActiveWaitlistEntries = 5

type WaitlistService interface {
	Join(ctx context.Context, userID, showtimeID string) (*dto.WaitlistEntryResponse, error)
	Cancel(ctx context.Context, userID, entryID string) error
	MyEntries(ctx context.Context, userID string) ([]dto.WaitlistEntryResponse, error)
}

type waitlistService struct {
	db   *gorm.DB
	repo *repository.WaitlistRepository
}

func NewWaitlistService(db *gorm.DB, repo *repository.WaitlistRepository) WaitlistService {
	return &waitlistService{db: db, repo: repo}
}

// Join is refused while the showtime still has an open or holdable seat —
// waitlisting only makes sense once it is genuinely sold out.
func (s *waitlistService) Join(ctx context.Context, userID, showtimeID string) (*dto.WaitlistEntryResponse, error) {
	available, err := s.repo.AvailableSeatCount(ctx, showtimeID)
	if err != nil {
		return nil, err
	}
	if available > 0 {
		return nil, apperrors.Conflict("this showtime still has seats available; no need to wait")
	}
	active, err := s.repo.CountActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if active >= maxActiveWaitlistEntries {
		return nil, apperrors.Conflict("too many active waitlist entries; cancel one before joining another")
	}
	entry := &models.WaitlistEntry{UserID: userID, ShowtimeID: showtimeID, Status: models.WaitlistWaiting}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.Create(tx, entry); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.Conflict("already waiting for this showtime")
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewWaitlistEntryResponse(entry)
	return &result, nil
}

func (s *waitlistService) Cancel(ctx context.Context, userID, entryID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		n, err := s.repo.Cancel(tx, entryID, userID)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.NotFound("waitlist entry not found or already settled")
		}
		return nil
	})
}

func (s *waitlistService) MyEntries(ctx context.Context, userID string) ([]dto.WaitlistEntryResponse, error) {
	rows, err := s.repo.MyEntries(ctx, userID)
	if err != nil {
		return nil, err
	}
	return dto.NewWaitlistEntryResponses(rows), nil
}
