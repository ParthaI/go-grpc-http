package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/parthasarathi/go-grpc-http/internal/user/model"
	"github.com/parthasarathi/go-grpc-http/internal/user/repository"
	"github.com/parthasarathi/go-grpc-http/pkg/auth"
)

type UserService struct {
	repo       repository.UserRepository
	jwtManager *auth.JWTManager
}

func NewUserService(repo repository.UserRepository, jwtManager *auth.JWTManager) *UserService {
	return &UserService{repo: repo, jwtManager: jwtManager}
}

func (s *UserService) Register(ctx context.Context, email, password, firstName, lastName string) (*model.User, error) {
	existing, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	authToken, err := generateAuthToken()
	if err != nil {
		return nil, fmt.Errorf("generate auth token: %w", err)
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.New().String(),
		Email:        email,
		PasswordHash: string(hash),
		FirstName:    firstName,
		LastName:     lastName,
		AuthToken:    authToken,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// generateAuthToken creates a cryptographically random 32-byte token, base64-encoded.
func generateAuthToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, string, int64, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", 0, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return "", "", 0, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", 0, fmt.Errorf("invalid credentials")
	}

	token, expiresAt, err := s.jwtManager.Generate(user.ID, user.Email, user.AuthToken)
	if err != nil {
		return "", "", 0, fmt.Errorf("generate token: %w", err)
	}

	return token, user.ID, expiresAt, nil
}

func (s *UserService) GetUser(ctx context.Context, id string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id, firstName, lastName string) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}
