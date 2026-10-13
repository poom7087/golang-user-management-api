package service

import (
	"context"
	"testing"
	"time"

	"github.com/poom7087/golang-user-management-api/internal/domain"
	"github.com/poom7087/golang-user-management-api/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// fakeUserRepository mocks MongoDB operations for unit tests.
type fakeUserRepository struct {
	users map[primitive.ObjectID]domain.User
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{
		users: make(map[primitive.ObjectID]domain.User),
	}
}

func (f *fakeUserRepository) Create(
	ctx context.Context,
	user *domain.User,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for _, existing := range f.users {
		if existing.Email == user.Email {
			return repository.ErrEmailExists
		}
	}

	if user.ID.IsZero() {
		user.ID = primitive.NewObjectID()
	}

	f.users[user.ID] = *user
	return nil
}

func (f *fakeUserRepository) GetByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, ok := f.users[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}

	result := user
	return &result, nil
}

func (f *fakeUserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	for _, user := range f.users {
		if user.Email == email {
			result := user
			return &result, nil
		}
	}

	return nil, repository.ErrUserNotFound
}

func (f *fakeUserRepository) List(
	ctx context.Context,
) ([]domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	result := make([]domain.User, 0, len(f.users))
	for _, user := range f.users {
		result = append(result, user)
	}

	return result, nil
}

func (f *fakeUserRepository) Update(
	ctx context.Context,
	id primitive.ObjectID,
	updates map[string]interface{},
) (*domain.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	user, ok := f.users[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}

	if name, ok := updates["name"].(string); ok {
		user.Name = name
	}

	if email, ok := updates["email"].(string); ok {
		user.Email = email
	}

	f.users[id] = user

	result := user
	return &result, nil
}

func (f *fakeUserRepository) Delete(
	ctx context.Context,
	id primitive.ObjectID,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, ok := f.users[id]; !ok {
		return repository.ErrUserNotFound
	}

	delete(f.users, id)
	return nil
}

func (f *fakeUserRepository) Count(
	ctx context.Context,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	return int64(len(f.users)), nil
}

func newTestUserService() (*UserService, *fakeUserRepository) {
	repo := newFakeUserRepository()

	userService := NewUserService(
		repo,
		"test-secret-key-for-unit-tests",
		24,
	)

	return userService, repo
}

func TestRegisterSuccess(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	input := domain.CreateUserInput{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "Password123!",
	}

	user, err := userService.Register(ctx, input)
	if err != nil {
		t.Fatalf("Register() returned unexpected error: %v", err)
	}

	if user == nil {
		t.Fatal("Register() returned nil user")
	}

	if user.Name != input.Name {
		t.Errorf("expected name %q, got %q", input.Name, user.Name)
	}

	if user.Email != input.Email {
		t.Errorf("expected email %q, got %q", input.Email, user.Email)
	}

	if user.Password == input.Password {
		t.Fatal("password was stored as plain text")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(input.Password),
	); err != nil {
		t.Errorf("stored password is not a valid hash: %v", err)
	}

	if user.ID.IsZero() {
		t.Error("expected generated user ID")
	}

	if user.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	input := domain.CreateUserInput{
		Name:     "Test User",
		Email:    "duplicate@example.com",
		Password: "Password123!",
	}

	if _, err := userService.Register(ctx, input); err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}

	_, err := userService.Register(ctx, input)
	if err == nil {
		t.Fatal("expected error when registering duplicate email")
	}
}

func TestRegisterInvalidInput(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	input := domain.CreateUserInput{
		Name:     "",
		Email:    "invalid-email",
		Password: "123",
	}

	_, err := userService.Register(ctx, input)
	if err == nil {
		t.Fatal("expected validation error for invalid input")
	}
}

func TestLoginSuccess(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	input := domain.CreateUserInput{
		Name:     "Login User",
		Email:    "login@example.com",
		Password: "Password123!",
	}

	if _, err := userService.Register(ctx, input); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	result, err := userService.Login(
		ctx,
		input.Email,
		input.Password,
	)
	if err != nil {
		t.Fatalf("Login() returned unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Login() returned nil result")
	}

	if result.Token == "" {
		t.Error("expected JWT token")
	}

	if result.User == nil {
		t.Fatal("expected user in login result")
	}

	if result.User.Email != input.Email {
		t.Errorf(
			"expected email %q, got %q",
			input.Email,
			result.User.Email,
		)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	input := domain.CreateUserInput{
		Name:     "Login User",
		Email:    "login@example.com",
		Password: "Password123!",
	}

	if _, err := userService.Register(ctx, input); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	_, err := userService.Login(
		ctx,
		input.Email,
		"WrongPassword123!",
	)
	if err == nil {
		t.Fatal("expected error for incorrect password")
	}
}

func TestCountUsers(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	initialCount, err := userService.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() returned unexpected error: %v", err)
	}

	if initialCount != 0 {
		t.Fatalf("expected 0 users initially, got %d", initialCount)
	}

	users := []domain.CreateUserInput{
		{
			Name:     "User One",
			Email:    "one@example.com",
			Password: "Password123!",
		},
		{
			Name:     "User Two",
			Email:    "two@example.com",
			Password: "Password123!",
		},
	}

	for _, input := range users {
		if _, err := userService.Register(ctx, input); err != nil {
			t.Fatalf("Register() failed: %v", err)
		}
	}

	count, err := userService.CountUsers(ctx)
	if err != nil {
		t.Fatalf("CountUsers() returned unexpected error: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 users, got %d", count)
	}
}

func TestGetUserByID(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	created, err := userService.Register(ctx, domain.CreateUserInput{
		Name:     "Find User",
		Email:    "find@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	found, err := userService.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID() returned unexpected error: %v", err)
	}

	if found.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID.Hex(), found.ID.Hex())
	}
}

func TestGetUserByInvalidID(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	nonExistentID := primitive.NewObjectID()

	_, err := userService.GetByID(ctx, nonExistentID)
	if err == nil {
		t.Fatal("expected error for non-existent user ID")
	}
}

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	created, err := userService.Register(ctx, domain.CreateUserInput{
		Name:     "Before Update",
		Email:    "before@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	newName := "After Update"
	updated, err := userService.Update(ctx, created.ID, domain.UpdateUserInput{
		Name: &newName,
	})
	if err != nil {
		t.Fatalf("Update() returned unexpected error: %v", err)
	}

	if updated.Name != newName {
		t.Errorf("expected name %q, got %q", newName, updated.Name)
	}

	if updated.Email != created.Email {
		t.Errorf("expected email to remain %q, got %q", created.Email, updated.Email)
	}
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	userService, _ := newTestUserService()

	created, err := userService.Register(ctx, domain.CreateUserInput{
		Name:     "Delete User",
		Email:    "delete@example.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	if err := userService.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete() returned unexpected error: %v", err)
	}

	_, err = userService.GetByID(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error when fetching deleted user")
	}
}

func TestExpiredContext(t *testing.T) {
	userService, _ := newTestUserService()

	ctx, cancel := context.WithDeadline(
		context.Background(),
		time.Now().Add(-time.Second),
	)
	defer cancel()

	_, err := userService.CountUsers(ctx)
	if err == nil {
		t.Fatal("expected error for expired context")
	}
}
