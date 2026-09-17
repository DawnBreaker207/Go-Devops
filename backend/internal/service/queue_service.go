package service

import (
	"context"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/cache"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// admitBatchSize: how many waiting customers are let in per Status poll
// (Phần 2.3 "vào theo lô N người/đợt", not one at a time).
const (
	admitBatchSize = 20
	queueEntryTTL  = time.Hour
	admitTokenTTL  = 10 * time.Minute // matches the hold TTL: being admitted is only useful long enough to hold
)

// QueueService is a gate in FRONT of Hold's transaction, not a replacement
// for its Postgres locking (Phần 3's invariant): it only decides admission
// order for a showtime an admin flagged high_traffic (queue_enabled). Redis
// unreachable -> every call fails open (nil *cache.Cache no-ops), so a queue
// outage never blocks selling tickets — Phần 2.3 "Bắt buộc fail-open".
type QueueService interface {
	// Join reserves this user's place in line; position is 0-based (0 = next).
	Join(ctx context.Context, showtimeID, userID string) (position int64, err error)
	// Status reports whether userID has been admitted; a first poll past the
	// batch boundary performs the admission itself.
	Status(ctx context.Context, showtimeID, userID string) (*dto.QueueStatusResponse, error)
	// IsAdmitted is the cheap check Hold uses; only consulted when queueing
	// is enabled for the showtime.
	IsAdmitted(ctx context.Context, showtimeID, userID string) (bool, error)
}

type queueService struct {
	cache *cache.Cache
}

func NewQueueService(c *cache.Cache) QueueService {
	return &queueService{cache: c}
}

func queueKey(showtimeID string) string         { return "queue:wait:" + showtimeID }
func admitKey(showtimeID, userID string) string { return "queue:admit:" + showtimeID + ":" + userID }

func (s *queueService) Join(ctx context.Context, showtimeID, userID string) (int64, error) {
	key := queueKey(showtimeID)
	if _, err := s.cache.ZAdd(ctx, key, float64(time.Now().UnixNano()), userID); err != nil {
		return 0, err
	}
	_ = s.cache.Expire(ctx, key, queueEntryTTL)
	rank, _, err := s.cache.ZRank(ctx, key, userID)
	if err != nil {
		return 0, err
	}
	return rank, nil
}

func (s *queueService) Status(ctx context.Context, showtimeID, userID string) (*dto.QueueStatusResponse, error) {
	admitted, err := s.IsAdmitted(ctx, showtimeID, userID)
	if err != nil {
		return nil, err
	}
	if admitted {
		return &dto.QueueStatusResponse{Admitted: true}, nil
	}
	rank, present, err := s.cache.ZRank(ctx, queueKey(showtimeID), userID)
	if err != nil {
		return nil, err
	}
	if !present {
		return nil, apperrors.NotFound("not in the queue for this showtime; POST /queue/join first")
	}
	if rank >= admitBatchSize {
		return &dto.QueueStatusResponse{Admitted: false, Position: rank}, nil
	}
	// Within the current batch: admit now, leave the waiting list.
	if err := s.cache.Set(ctx, admitKey(showtimeID, userID), "1", admitTokenTTL); err != nil {
		return nil, err
	}
	if err := s.cache.ZRem(ctx, queueKey(showtimeID), userID); err != nil {
		return nil, err
	}
	return &dto.QueueStatusResponse{Admitted: true}, nil
}

func (s *queueService) IsAdmitted(ctx context.Context, showtimeID, userID string) (bool, error) {
	_, ok, err := s.cache.Get(ctx, admitKey(showtimeID, userID))
	return ok, err
}
