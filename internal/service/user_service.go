package service

import (
	"context"
	"fmt"

	"github.com/aarondever/go-gin-template/internal/repository"
	"github.com/aarondever/go-gin-template/internal/worker"
	"github.com/aarondever/go-gin-template/pkg/logger"
)

type UserService struct {
	userRepo   *repository.UserRepository
	workerPool *worker.Pool
}

func NewUserService(userRepo *repository.UserRepository, workerPool *worker.Pool) *UserService {
	return &UserService{
		userRepo:   userRepo,
		workerPool: workerPool,
	}
}

func (s *UserService) CreateUser(ctx context.Context, username, email string) (*repository.User, error) {
	user := &repository.User{
		Username: username,
		Email:    email,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Submit async job to worker pool (e.g., send welcome email)
	s.workerPool.Submit(worker.Job{
		Type: "send_welcome_email",
		Payload: map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
		},
		Handler: func(ctx context.Context, job worker.Job) error {
			logger.Info("Sending welcome email", "user_id", job.Payload["user_id"], "email", job.Payload["email"])
			// Simulate email sending
			// emailService.SendWelcome(job.Payload["email"].(string))
			return nil
		},
	})

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, id int64) (*repository.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]repository.User, error) {
	users, err := s.userRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id int64, username, email string) (*repository.User, error) {
	user := &repository.User{
		ID:       id,
		Username: username,
		Email:    email,
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id int64) error {
	if err := s.userRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
