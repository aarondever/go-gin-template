package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aarondever/go-gin-template/internal/config"
	"github.com/aarondever/go-gin-template/internal/database"
	"github.com/aarondever/go-gin-template/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	cfg *config.Config
	db  *database.Database
}

func NewUserService(cfg *config.Config, db *database.Database) *UserService {
	return &UserService{
		cfg: cfg,
		db:  db,
	}
}

func (service *UserService) CreateUser(ctx context.Context, params models.UserParams) (*database.User, error) {
	hash, err := generatePasswordHash(params.Password)
	if err != nil {
		return nil, err
	}

	user, err := service.db.Queries.CreateUser(ctx, database.CreateUserParams{
		Username:     params.Username,
		PasswordHash: hash,
	})
	if err != nil {
		return nil, checkPgErr(err)
	}

	return user, nil
}

func (service *UserService) UpdateUser(ctx context.Context, params models.UpdateUserParams) (*database.User, error) {
	user, err := service.db.Queries.UpdateUser(ctx, database.UpdateUserParams{
		ID:       pgtype.UUID{Bytes: params.ID, Valid: true},
		Username: params.Username,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, checkPgErr(err)
	}

	return user, nil
}

func (service *UserService) Login(ctx context.Context, params models.UserParams) (*database.User, error) {
	user, err := service.GetUserByUsername(ctx, params.Username)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, models.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, models.ErrInvalidCredentials
		}

		slog.Error("Failed to compare password hash", "error", err)
		return nil, err
	}

	return user, nil
}

func (service *UserService) GetUser(ctx context.Context, id uuid.UUID) (*database.User, error) {
	user, err := service.db.Queries.GetUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, checkPgErr(err)
	}

	return user, nil
}

func (service *UserService) GetUserByUsername(ctx context.Context, username string) (*database.User, error) {
	user, err := service.db.Queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, checkPgErr(err)
	}

	return user, nil
}

func (service *UserService) GetUsers(ctx context.Context, params database.GetUsersParams) ([]*database.User, error) {
	return service.db.Queries.GetUsers(ctx, params)
}

func (service *UserService) GetUserPagination(
	ctx context.Context,
	params database.GetUsersParams,
) (*models.PaginationResult[*database.User], error) {
	users, err := service.GetUsers(ctx, params)
	if err != nil {
		return nil, err
	}

	userCount, err := service.db.Queries.GetUserCount(ctx)
	if err != nil {
		return nil, err
	}

	result := &models.PaginationResult[*database.User]{
		Data:  users,
		Total: userCount,
	}

	return result, nil
}

// generatePasswordHash hashes the given password using bcrypt.
func generatePasswordHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("Failed to generate password hash", "error", err)
		return "", err
	}

	return string(bytes), nil
}

// checkPgErr checks for PostgresSQL errors and maps them to application-specific errors.
func checkPgErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		// 23505 is unique_violation
		case "23505":
			return models.ErrDuplicateUser
		default:
			return err
		}
	}

	return err
}
