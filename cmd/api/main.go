package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/poom7087/golang-user-management-api/internal/config"
	"github.com/poom7087/golang-user-management-api/internal/database"
	"github.com/poom7087/golang-user-management-api/internal/middleware"
	"github.com/poom7087/golang-user-management-api/internal/repository"
	"github.com/poom7087/golang-user-management-api/internal/routes"
	"github.com/poom7087/golang-user-management-api/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required; set it in .env")
	}

	// Connect to MongoDB.
	startupCtx, startupCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	mongoClient, err := database.Connect(startupCtx, cfg.MongoURI)
	startupCancel()

	if err != nil {
		log.Fatalf("MongoDB initialization failed: %v", err)
	}

	// Ensure MongoDB is disconnected on exit.
	defer func() {
		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("MongoDB disconnect error: %v", err)
		}
	}()

	// Initialize MongoDB database and repository.
	mongoDB := mongoClient.Database(cfg.MongoDBName)
	userRepo := repository.NewMongoUserRepository(mongoDB)

	// Ensure the unique email index exists.
	indexCtx, indexCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	err = userRepo.EnsureIndexes(indexCtx)
	indexCancel()

	if err != nil {
		log.Fatalf("MongoDB index initialization failed: %v", err)
	}

	log.Println("MongoDB connected and user indexes initialized")

	// Initialize application service.
	userService := service.NewUserService(
		userRepo,
		cfg.JWTSecret,
		cfg.JWTExpiresIn,
	)

	// Initialize context for the background task.
	backgroundCtx, stopBackground := context.WithCancel(
		context.Background(),
	)

	// Log total users every 10 seconds.
	backgroundDone := make(chan struct{})

	go func(ctx context.Context) {
		defer close(backgroundDone)

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("User count background task stopped")
				return

			case <-ticker.C:
				countCtx, cancel := context.WithTimeout(
					ctx,
					5*time.Second,
				)

				count, err := userService.CountUsers(countCtx)
				cancel()

				if err != nil {
					if ctx.Err() != nil {
						return
					}

					log.Printf("Failed to count users: %v", err)
					continue
				}

				log.Printf("Total users in database: %d", count)
			}
		}
	}(backgroundCtx)

	// Initialize Fiber application.
	app := fiber.New(fiber.Config{
		AppName: "User Management API",
	})

	// Register global middleware.
	app.Use(middleware.Logger())

	// Register API routes.
	routes.Register(app, userService, cfg.JWTSecret)

	// Start HTTP server.
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- app.Listen(":" + cfg.Port)
	}()

	log.Printf("API starting on port %s", cfg.Port)

	// Wait for shutdown signal or server error.
	signalCtx, stopSignal := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	select {
	case <-signalCtx.Done():
		log.Println("Shutdown signal received")

	case err := <-serverErr:
		if err != nil {
			log.Printf("HTTP server stopped: %v", err)
		}
	}

	stopSignal()

	// Stop the background task before disconnecting MongoDB.
	stopBackground()

	select {
	case <-backgroundDone:
	case <-time.After(6 * time.Second):
		log.Println("Timed out waiting for background task to stop")
	}

	// Gracefully shut down HTTP server.
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	shutdownCancel()

	log.Println("API shutdown complete")
}
