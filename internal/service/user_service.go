package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/poom7087/golang-user-management-api/internal/domain"
	"github.com/poom7087/golang-user-management-api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type UserService struct {
	repo       repository.UserRepository
	jwtSecret  string
	tokenHours int
}

func NewUserService(
	repo repository.UserRepository,
	jwtSecret string,
	tokenHours int,
) *UserService {
	return &UserService{
		repo:       repo,
		jwtSecret:  jwtSecret,
		tokenHours: tokenHours,
	}
}

type AuthResult struct {
	User  *domain.User `json:"user"`
	Token string       `json:"token"`
}

func validateEmail(email string) bool {
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email && len(email) <= 254
}

func (s *UserService) Register(
	ctx context.Context,
	input domain.CreateUserInput,
) (*domain.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if name == "" || len(name) > 100 || !validateEmail(email) {
		return nil, ErrInvalidInput
	}

	if len(input.Password) < 8 || len(input.Password) > 72 {
		return nil, ErrInvalidInput
	}

	_, err := s.repo.GetByEmail(ctx, email)
	if err == nil {
		return nil, repository.ErrEmailExists
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(input.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Name:      name,
		Email:     email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(
	ctx context.Context,
	email, password string,
) (*AuthResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	); err != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now()
	expiresAt := now.Add(time.Duration(s.tokenHours) * time.Hour)

	claims := jwt.RegisteredClaims{
		Subject:   user.ID.Hex(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		User:  user,
		Token: signedToken,
	}, nil
}

func (s *UserService) GetByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(
	ctx context.Context,
) ([]domain.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) Update(
	ctx context.Context,
	id primitive.ObjectID,
	input domain.UpdateUserInput,
) (*domain.User, error) {
	updates := make(map[string]interface{})

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" || len(name) > 100 {
			return nil, ErrInvalidInput
		}
		updates["name"] = name
	}

	if input.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*input.Email))
		if !validateEmail(email) {
			return nil, ErrInvalidInput
		}
		updates["email"] = email
	}

	if len(updates) == 0 {
		return nil, ErrInvalidInput
	}

	return s.repo.Update(ctx, id, updates)
}

func (s *UserService) Delete(
	ctx context.Context,
	id primitive.ObjectID,
) error {
	return s.repo.Delete(ctx, id)
}

func ParseUserID(id string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(id)
}

// CountUsers returns the total number of users.
func (s *UserService) CountUsers(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}
