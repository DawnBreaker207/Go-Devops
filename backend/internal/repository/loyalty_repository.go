package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"gorm.io/gorm"
)

type LoyaltyRepository struct {
	db *gorm.DB
}

func NewLoyaltyRepository(db *gorm.DB) *LoyaltyRepository {
	return &LoyaltyRepository{db: db}
}

func (r *LoyaltyRepository) Balance(ctx context.Context, userID string) (int64, error) {
	var balance int64
	err := r.db.WithContext(ctx).Raw(`SELECT balance FROM user_points WHERE user_id = ?`, userID).Scan(&balance).Error
	return balance, err
}

func (r *LoyaltyRepository) Transactions(ctx context.Context, userID string, limit, offset int) ([]models.PointTransaction, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.PointTransaction{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []models.PointTransaction
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

// AddBalance credits points and appends the earn log row, in the caller's transaction.
func (r *LoyaltyRepository) AddBalance(tx *gorm.DB, userID string, points int64, bookingID string) error {
	if points <= 0 {
		return nil
	}
	if err := tx.Exec(`
		INSERT INTO user_points (user_id, balance, updated_at) VALUES (?, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE SET balance = user_points.balance + EXCLUDED.balance, updated_at = NOW()`,
		userID, points).Error; err != nil {
		return fmt.Errorf("add points balance: %w", err)
	}
	txn := &models.PointTransaction{UserID: userID, Amount: points, Type: models.PointEarn, ReferenceBookingID: &bookingID}
	return tx.Create(txn).Error
}

// IncrMilestone advances the ticket counter toward the next loyalty voucher
// and returns how many milestones were just crossed (0, 1, or more for a
// single large multi-seat booking), wrapping the remainder — Phần 2.1
// "LẶP LẠI theo chu kỳ".
func (r *LoyaltyRepository) IncrMilestone(tx *gorm.DB, userID string, ticketCount, every int) (crossed int, err error) {
	var newCount int
	err = tx.Raw(`
		INSERT INTO user_points (user_id, balance, tickets_toward_milestone, updated_at) VALUES (?, 0, ?, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			tickets_toward_milestone = user_points.tickets_toward_milestone + EXCLUDED.tickets_toward_milestone,
			updated_at = NOW()
		RETURNING tickets_toward_milestone`, userID, ticketCount).Scan(&newCount).Error
	if err != nil {
		return 0, fmt.Errorf("incr milestone counter: %w", err)
	}
	for newCount >= every {
		newCount -= every
		crossed++
	}
	if crossed > 0 {
		if err := tx.Exec(`UPDATE user_points SET tickets_toward_milestone = ? WHERE user_id = ?`, newCount, userID).Error; err != nil {
			return 0, fmt.Errorf("wrap milestone counter: %w", err)
		}
	}
	return crossed, nil
}

// SpendPoints is the CAS that reserves points for a redemption: it only
// succeeds while the balance is still enough (Phần 2.1 TOCTOU fix).
func (r *LoyaltyRepository) SpendPoints(tx *gorm.DB, userID string, cost int64) (int64, error) {
	res := tx.Exec(`UPDATE user_points SET balance = balance - ?, updated_at = NOW() WHERE user_id = ? AND balance >= ?`,
		cost, userID, cost)
	if res.Error != nil {
		return 0, fmt.Errorf("spend points: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *LoyaltyRepository) RefundPoints(tx *gorm.DB, userID string, amount int64, rewardID string) error {
	if err := tx.Exec(`UPDATE user_points SET balance = balance + ?, updated_at = NOW() WHERE user_id = ?`, amount, userID).Error; err != nil {
		return fmt.Errorf("refund points: %w", err)
	}
	txn := &models.PointTransaction{UserID: userID, Amount: amount, Type: models.PointRefundReversal, ReferenceRewardID: &rewardID}
	return tx.Create(txn).Error
}

func (r *LoyaltyRepository) CreateRedeemTransaction(tx *gorm.DB, userID string, cost int64, rewardID string) error {
	txn := &models.PointTransaction{UserID: userID, Amount: -cost, Type: models.PointRedeem, ReferenceRewardID: &rewardID}
	return tx.Create(txn).Error
}

// --- Rewards ---

func (r *LoyaltyRepository) ListRewards(ctx context.Context, activeOnly bool) ([]models.Reward, error) {
	q := r.db.WithContext(ctx).Order("points_cost")
	if activeOnly {
		q = q.Where("active = true")
	}
	var rows []models.Reward
	err := q.Find(&rows).Error
	return rows, err
}

func (r *LoyaltyRepository) FindReward(ctx context.Context, tx *gorm.DB, id string) (*models.Reward, error) {
	conn := r.db.WithContext(ctx)
	if tx != nil {
		conn = tx
	}
	var reward models.Reward
	if err := conn.First(&reward, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reward, nil
}

func (r *LoyaltyRepository) CreateReward(tx *gorm.DB, reward *models.Reward) error {
	return tx.Create(reward).Error
}

func (r *LoyaltyRepository) UpdateReward(tx *gorm.DB, reward *models.Reward) error {
	return tx.Model(reward).Select(
		"name", "points_cost", "stock_quantity", "voucher_template_id", "active", "updated_at",
	).Updates(reward).Error
}

// DecrStock is the CAS for a limited reward; a NULL stock_quantity (unlimited) always succeeds.
func (r *LoyaltyRepository) DecrStock(tx *gorm.DB, rewardID string) (int64, error) {
	res := tx.Exec(`UPDATE rewards SET stock_quantity = stock_quantity - 1
		WHERE id = ? AND (stock_quantity IS NULL OR stock_quantity > 0)`, rewardID)
	if res.Error != nil {
		return 0, fmt.Errorf("decr reward stock: %w", res.Error)
	}
	return res.RowsAffected, nil
}

func (r *LoyaltyRepository) CreateRedemption(tx *gorm.DB, red *models.RewardRedemption) error {
	return tx.Create(red).Error
}

func (r *LoyaltyRepository) FindRedemptionByCode(ctx context.Context, code string) (*models.RewardRedemption, error) {
	var red models.RewardRedemption
	if err := r.db.WithContext(ctx).First(&red, "code = ?", code).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &red, nil
}

func (r *LoyaltyRepository) MarkDelivered(tx *gorm.DB, code string) (int64, error) {
	res := tx.Exec(`UPDATE reward_redemptions SET status = ?, delivered_at = NOW()
		WHERE code = ? AND status = ?`, models.RewardRedemptionDelivered, code, models.RewardRedemptionPendingPickup)
	return res.RowsAffected, res.Error
}
