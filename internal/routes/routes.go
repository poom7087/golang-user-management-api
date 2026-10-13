package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/poom7087/golang-user-management-api/internal/handler"
	"github.com/poom7087/golang-user-management-api/internal/middleware"
	"github.com/poom7087/golang-user-management-api/internal/service"
)

func Register(
	app *fiber.App,
	userService *service.UserService,
	jwtSecret string,
) {
	userHandler := handler.NewUserHandler(userService)

	// Public endpoints
	app.Get("/health", handler.Health)

	auth := app.Group("/api/v1/auth")
	auth.Post("/register", userHandler.Register)
	auth.Post("/login", userHandler.Login)

	// All endpoints in this group require a valid JWT.
	protected := app.Group(
		"/api/v1",
		middleware.Auth(jwtSecret),
	)

	// Temporary endpoint for testing authentication.
	protected.Get("/me", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"user_id": c.Locals("userID"),
		})
	})

	// User management endpoints.
	users := protected.Group("/users")
	users.Post("/", userHandler.Create)
	users.Get("/", userHandler.List)
	users.Get("/:id", userHandler.GetByID)
	users.Patch("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)
}
