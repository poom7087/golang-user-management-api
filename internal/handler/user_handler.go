package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/poom7087/golang-user-management-api/internal/domain"
	"github.com/poom7087/golang-user-management-api/internal/repository"
	"github.com/poom7087/golang-user-management-api/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{service: userService}
}

func requestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (h *UserHandler) Register(c *fiber.Ctx) error {
	var input domain.CreateUserInput

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	user, err := h.service.Register(ctx, input)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Name, valid email, and password of 8-72 bytes are required",
			})
		case errors.Is(err, repository.ErrEmailExists):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"message": "Email already exists",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Could not register user",
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "User registered successfully",
		"data":    user,
	})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *UserHandler) Login(c *fiber.Ctx) error {
	var input loginRequest

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	result, err := h.service.Login(ctx, input.Email, input.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "Invalid email or password",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not authenticate user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Login successful",
		"data":    result,
	})
}

func (h *UserHandler) Create(c *fiber.Ctx) error {
	return h.Register(c)
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	ctx, cancel := requestContext(c)
	defer cancel()

	users, err := h.service.List(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": "Could not retrieve users",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": users,
	})
}

func (h *UserHandler) GetByID(c *fiber.Ctx) error {
	id, err := service.ParseUserID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID",
		})
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	user, err := h.service.GetByID(ctx, id)
	if err != nil {
		return handleUserError(c, err, "Could not retrieve user")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": user,
	})
}

func (h *UserHandler) Update(c *fiber.Ctx) error {
	id, err := service.ParseUserID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID",
		})
	}

	var input domain.UpdateUserInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid request body",
		})
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	user, err := h.service.Update(ctx, id, input)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"message": "Provide a valid name or email",
			})
		}
		return handleUserError(c, err, "Could not update user")
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "User updated successfully",
		"data":    user,
	})
}

func (h *UserHandler) Delete(c *fiber.Ctx) error {
	id, err := service.ParseUserID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"message": "Invalid user ID",
		})
	}

	ctx, cancel := requestContext(c)
	defer cancel()

	if err := h.service.Delete(ctx, id); err != nil {
		return handleUserError(c, err, "Could not delete user")
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func handleUserError(c *fiber.Ctx, err error, fallback string) error {
	switch {
	case errors.Is(err, repository.ErrUserNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"message": "User not found",
		})
	case errors.Is(err, repository.ErrEmailExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"message": "Email already exists",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": fallback,
		})
	}
}
