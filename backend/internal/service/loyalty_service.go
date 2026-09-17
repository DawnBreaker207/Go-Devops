package service

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

// pointsPerVND and milestoneTickets are the two concrete numbers
// ADVANCED_FEATURES_DISCUSSION.md Phần 2.1 gives as the settled example
// (Lotte L.Point 1.000đ=1 điểm; "cứ đủ 10 vé kế tiếp lại được 1 voucher").
const (
	pointsPerVND     = 1000
	milestoneTickets = 10
	// milestoneVoucherValue/TTL are this system's own default reward for a
	// milestone (not specified by the discussion doc — a concrete choice had
	// to be made somewhere); change here if the business picks a different amount.
	milestoneVoucherValue = 50000
	milestoneVoucherTTL   = 90 * 24 * time.Hour
)

type LoyaltyService interface {
	Balance(ctx context.Context, userID string) (dto.PointsBalanceResponse, error)
	Transactions(ctx context.Context, userID string, query dto.PageQuery) ([]dto.PointTransactionResponse, int64, error)

	ListRewards(ctx context.Context, activeOnly bool) ([]dto.RewardResponse, error)
	CreateReward(ctx context.Context, req dto.RewardRequest) (*dto.RewardResponse, error)
	UpdateReward(ctx context.Context, id string, req dto.UpdateRewardRequest) (*dto.RewardResponse, error)
	Redeem(ctx context.Context, userID, rewardID string) (*dto.RewardRedemptionResponse, error)
	MarkDelivered(ctx context.Context, code string) error

	// EarnForBooking credits points and advances the milestone counter for a
	// just-confirmed, just-paid booking (online or counter). Must run inside
	// the same transaction that confirms the booking. userID may be empty
	// (a counter sale with no account earns nothing, Phần 2.1 still applies
	// "trên MỌI kênh" only to accounts that exist).
	EarnForBooking(ctx context.Context, tx *gorm.DB, userID string, amountVND int64, ticketCount int, bookingID string) error
}

type loyaltyService struct {
	db       *gorm.DB
	repo     *repository.LoyaltyRepository
	vouchers *repository.VoucherRepository
	ledger   *repository.LedgerRepository
}

func NewLoyaltyService(db *gorm.DB, repo *repository.LoyaltyRepository, vouchers *repository.VoucherRepository, ledger *repository.LedgerRepository) LoyaltyService {
	return &loyaltyService{db: db, repo: repo, vouchers: vouchers, ledger: ledger}
}

func (s *loyaltyService) Balance(ctx context.Context, userID string) (dto.PointsBalanceResponse, error) {
	balance, err := s.repo.Balance(ctx, userID)
	if err != nil {
		return dto.PointsBalanceResponse{}, err
	}
	return dto.PointsBalanceResponse{Balance: balance}, nil
}

func (s *loyaltyService) Transactions(ctx context.Context, userID string, query dto.PageQuery) ([]dto.PointTransactionResponse, int64, error) {
	rows, total, err := s.repo.Transactions(ctx, userID, query.PageSize, query.Offset())
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.PointTransactionResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, dto.NewPointTransactionResponse(r))
	}
	return out, total, nil
}

func (s *loyaltyService) ListRewards(ctx context.Context, activeOnly bool) ([]dto.RewardResponse, error) {
	rows, err := s.repo.ListRewards(ctx, activeOnly)
	if err != nil {
		return nil, err
	}
	return dto.NewRewardResponses(rows), nil
}

func (s *loyaltyService) CreateReward(ctx context.Context, req dto.RewardRequest) (*dto.RewardResponse, error) {
	reward := &models.Reward{Type: req.Type, Name: strings.TrimSpace(req.Name), PointsCost: req.PointsCost, StockQuantity: req.StockQuantity, Active: true}
	if req.VoucherTemplateID != "" {
		reward.VoucherTemplateID = &req.VoucherTemplateID
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateReward(tx, reward); err != nil {
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = reward.ID
			rec.After = map[string]any{"name": reward.Name, "points_cost": reward.PointsCost}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewRewardResponse(reward)
	return &result, nil
}

func (s *loyaltyService) UpdateReward(ctx context.Context, id string, req dto.UpdateRewardRequest) (*dto.RewardResponse, error) {
	var reward *models.Reward
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := s.repo.FindReward(ctx, nil, id)
		if err != nil {
			return err
		}
		if current == nil {
			return apperrors.ErrRewardNotFound
		}
		if req.Name != nil {
			current.Name = strings.TrimSpace(*req.Name)
		}
		if req.PointsCost != nil {
			current.PointsCost = *req.PointsCost
		}
		if req.StockQuantity != nil {
			current.StockQuantity = req.StockQuantity
		}
		if req.VoucherTemplateID != nil {
			current.VoucherTemplateID = req.VoucherTemplateID
		}
		if req.Active != nil {
			current.Active = *req.Active
		}
		if err := s.repo.UpdateReward(tx, current); err != nil {
			return err
		}
		reward = current
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = id
			rec.After = map[string]any{"points_cost": reward.PointsCost, "active": reward.Active}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewRewardResponse(reward)
	return &result, nil
}

// Redeem: not refundable/cancelable (I8 "không hoàn" policy applies here
// too, Phần 2.1) — points spent, stock (if limited) decremented, and a
// pickup code issued, all atomically.
func (s *loyaltyService) Redeem(ctx context.Context, userID, rewardID string) (*dto.RewardRedemptionResponse, error) {
	var red *models.RewardRedemption
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		reward, err := s.repo.FindReward(ctx, tx, rewardID)
		if err != nil {
			return err
		}
		if reward == nil {
			return apperrors.ErrRewardNotFound
		}
		if !reward.Active {
			return apperrors.ErrRewardInactive
		}
		n, err := s.repo.SpendPoints(tx, userID, reward.PointsCost)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.ErrInsufficientPoints
		}
		if reward.StockQuantity != nil {
			n, err := s.repo.DecrStock(tx, rewardID)
			if err != nil {
				return err
			}
			if n == 0 {
				return apperrors.ErrRewardOutOfStock
			}
		}
		status := models.RewardRedemptionPendingPickup
		if reward.Type == models.RewardVoucher {
			status = models.RewardRedemptionDelivered // a voucher reward is delivered digitally, at once
		}
		redemption := &models.RewardRedemption{
			RewardID: rewardID, UserID: userID, PointsSpent: reward.PointsCost,
			Code: randomHex(8), Status: status, IssuedVoucherID: reward.VoucherTemplateID,
		}
		if status == models.RewardRedemptionDelivered {
			now := time.Now()
			redemption.DeliveredAt = &now
		}
		if err := s.repo.CreateRedemption(tx, redemption); err != nil {
			return err
		}
		if err := s.repo.CreateRedeemTransaction(tx, userID, reward.PointsCost, rewardID); err != nil {
			return err
		}
		ledger := &models.LedgerEntry{
			Type: models.LedgerRewardRedeemed, Amount: reward.PointsCost,
			ReferenceTable: "reward_redemptions", ReferenceID: redemption.ID, UserID: &userID,
		}
		if err := s.ledger.Write(tx, ledger); err != nil {
			return err
		}
		red = redemption
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = redemption.ID
			rec.After = map[string]any{"reward_id": rewardID, "points_spent": reward.PointsCost, "code": redemption.Code}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewRewardRedemptionResponse(red)
	return &result, nil
}

func (s *loyaltyService) MarkDelivered(ctx context.Context, code string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		n, err := s.repo.MarkDelivered(tx, code)
		if err != nil {
			return err
		}
		if n == 0 {
			return apperrors.NotFound("redemption code not found or already delivered")
		}
		return nil
	})
}

func (s *loyaltyService) EarnForBooking(ctx context.Context, tx *gorm.DB, userID string, amountVND int64, ticketCount int, bookingID string) error {
	if userID == "" || amountVND <= 0 || ticketCount <= 0 {
		return nil
	}
	points := amountVND / pointsPerVND
	if points > 0 {
		if err := s.repo.AddBalance(tx, userID, points, bookingID); err != nil {
			return err
		}
	}
	crossed, err := s.repo.IncrMilestone(tx, userID, ticketCount, milestoneTickets)
	if err != nil {
		return err
	}
	for i := 0; i < crossed; i++ {
		if err := s.issueMilestoneVoucher(tx, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *loyaltyService) issueMilestoneVoucher(tx *gorm.DB, userID string) error {
	now := time.Now()
	v := &models.Voucher{
		Code: "LOYAL-" + randomHex(6), DiscountType: models.VoucherDiscountFixed, DiscountValue: milestoneVoucherValue,
		MinOrderAmount: 0, StartsAt: now, EndsAt: now.Add(milestoneVoucherTTL),
		MaxUsage: 1, MaxUsagePerUser: 1, ApplyScope: models.VoucherScopeAll,
		IsSystemIssued: true, AssignedUserID: &userID, Status: models.VoucherActive,
	}
	return s.vouchers.Create(tx, v)
}
