package database

import (
	"context"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// SeedAdmin creates the first admin account if the users table is empty.
// Only runs in non-production environments to allow login right after a clone.
func SeedAdmin(ctx context.Context, db *gorm.DB, email, password string) error {
	var count int64
	if err := db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash seed password: %w", err)
	}

	admin := &models.User{
		Email:    email,
		Password: string(hashed),
		FullName: "Administrator",
		Role:     models.RoleAdmin,
	}
	if err := db.WithContext(ctx).Create(admin).Error; err != nil {
		return fmt.Errorf("create seed admin: %w", err)
	}

	// Email stays out of the log (app.admin_email in config tells which one).
	logger.Warn("seeded default admin account, doi mat khau ngay sau lan dang nhap dau tien")
	return nil
}
