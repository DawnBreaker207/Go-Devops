package service

import (
	"context"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

var phonePattern = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

type UserService interface {
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
	List(ctx context.Context, query dto.UserListQuery) ([]dto.UserResponse, int64, error)
	Create(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error)
	Update(ctx context.Context, actorID, userID string, req dto.UpdateUserRequest) (*dto.UserResponse, error)
	UpdateProfile(ctx context.Context, userID string, req dto.UpdateProfileRequest) (*dto.UserResponse, error)
	DeleteMe(ctx context.Context, userID string) error
}

type userService struct {
	db       *gorm.DB
	userRepo repository.UserRepository
	onChange []func(userID string)
}

// NewUserService: onChange callbacks run after an account's active flag or role
// changed (e.g. AccountStatusCache.Invalidate).
func NewUserService(db *gorm.DB, userRepo repository.UserRepository, onChange ...func(userID string)) UserService {
	return &userService{db: db, userRepo: userRepo, onChange: onChange}
}

// UpdateProfile changes the caller's own name and phone; an empty phone clears it.
func (s *userService) UpdateProfile(ctx context.Context, userID string, req dto.UpdateProfileRequest) (*dto.UserResponse, error) {
	fullName := strings.TrimSpace(req.FullName)
	phone := strings.TrimSpace(req.Phone)
	if phone != "" && !phonePattern.MatchString(phone) {
		return nil, apperrors.Validation("phone must be 8 to 15 digits, optionally prefixed with +")
	}
	var user *models.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		u, err := s.userRepo.LockByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if u == nil {
			return apperrors.ErrUserNotFound
		}
		if err := s.userRepo.UpdateProfile(ctx, tx, userID, fullName, phone); err != nil {
			return err
		}
		u.FullName, u.Phone = fullName, phone
		user = u
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = userID
			rec.After = map[string]any{"full_name": user.FullName, "phone": user.Phone}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}

// DeleteMe erases the caller's account under the right to erasure. Confirmed
// tickets still to come come first, so they block the delete (409); afterwards
// personal fields are scrubbed, sessions dropped and unpaid holds expired.
func (s *userService) DeleteMe(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		user, err := s.userRepo.LockByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if user == nil {
			return apperrors.ErrUserNotFound
		}

		var owed int64
		if err := tx.Model(&models.Booking{}).
			Joins("JOIN showtimes ON showtimes.id = bookings.showtime_id").
			Where("bookings.user_id = ? AND bookings.status = ? AND showtimes.end_at > NOW()",
				userID, models.BookingConfirmed).Count(&owed).Error; err != nil {
			return err
		}
		if owed > 0 {
			return apperrors.ErrAccountHoldsTickets
		}

		if err := tx.Model(&models.Booking{}).
			Where("user_id = ? AND status = ?", userID, models.BookingPending).
			Update("status", models.BookingExpired).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.PasswordResetToken{}).Error; err != nil {
			return err
		}

		anonEmail := "deleted-" + userID + "@anon.invalid"
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]any{
			"email": anonEmail, "full_name": "", "phone": "",
			"password": anonEmail + "!", "role": models.RoleCustomer,
			"active": false, "accepted_terms_version": 0,
		}).Error; err != nil {
			return err
		}

		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = userID
			rec.Before = map[string]any{"email": user.Email}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}

func (s *userService) List(ctx context.Context, query dto.UserListQuery) ([]dto.UserResponse, int64, error) {
	users, total, err := s.userRepo.List(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	return dto.NewUserResponses(users), total, nil
}

func (s *userService) Create(ctx context.Context, req dto.CreateUserRequest) (*dto.UserResponse, error) {
	email := normalizeEmail(req.Email)
	exists, err := s.userRepo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperrors.ErrEmailAlreadyExists
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperrors.Internal("cannot hash password").Wrap(err)
	}
	user := &models.User{
		Email:    email,
		Password: string(hashed),
		FullName: strings.TrimSpace(req.FullName),
		Role:     req.Role,
		Active:   true,
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.userRepo.CreateTx(ctx, tx, user); err != nil {
			if apperrors.IsUniqueViolation(err) {
				return apperrors.ErrEmailAlreadyExists
			}
			return err
		}
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = user.ID
			rec.After = map[string]any{"email": user.Email, "full_name": user.FullName, "role": user.Role}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}

// Update locks active admins first, in id order, so two admins acting on each other
// can not both win.
func (s *userService) Update(ctx context.Context, actorID, userID string, req dto.UpdateUserRequest) (*dto.UserResponse, error) {
	if req.Active == nil && req.Role == nil {
		return nil, apperrors.Validation("nothing to update: send active and/or role")
	}
	var user *models.User
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		admins, err := s.userRepo.LockActiveAdmins(ctx, tx)
		if err != nil {
			return err
		}
		target, err := s.userRepo.LockByID(ctx, tx, userID)
		if err != nil {
			return err
		}
		if target == nil {
			return apperrors.ErrUserNotFound
		}

		newActive, newRole := target.Active, target.Role
		if req.Active != nil {
			newActive = *req.Active
		}
		if req.Role != nil {
			newRole = *req.Role
		}
		if target.ID == actorID {
			if target.Active && !newActive {
				return apperrors.ErrCannotLockSelf
			}
			if newRole != target.Role {
				return apperrors.ErrCannotDemoteSelf
			}
		}
		wasActiveAdmin := target.Active && target.Role == models.RoleAdmin
		staysActiveAdmin := newActive && newRole == models.RoleAdmin
		if wasActiveAdmin && !staysActiveAdmin && len(admins) <= 1 {
			return apperrors.ErrLastAdmin
		}

		before := map[string]any{"active": target.Active, "role": target.Role}
		if newActive != target.Active {
			if err := s.userRepo.SetActive(ctx, tx, target.ID, newActive); err != nil {
				return err
			}
		}
		if newRole != target.Role {
			if err := s.userRepo.SetRole(ctx, tx, target.ID, newRole); err != nil {
				return err
			}
		}
		target.Active, target.Role = newActive, newRole
		user = target
		if rec, ok := audit.FromContext(ctx); ok {
			rec.ResourceID = target.ID
			rec.Before = before
			rec.After = map[string]any{"active": newActive, "role": newRole}
			return audit.In(ctx, tx, rec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, fn := range s.onChange {
		fn(userID)
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}
