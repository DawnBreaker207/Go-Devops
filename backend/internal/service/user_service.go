package service

import (
	"context"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

// UserService xu ly nghiep vu lien quan den nguoi dung.
type UserService interface {
	GetByID(ctx context.Context, id string) (*dto.UserResponse, error)
}

type userService struct {
	userRepo repository.UserRepository
}

// NewUserService tao UserService.
func NewUserService(userRepo repository.UserRepository) UserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetByID(ctx context.Context, id string) (*dto.UserResponse, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	result := dto.NewUserResponse(user)
	return &result, nil
}
