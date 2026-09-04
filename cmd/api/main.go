package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"api-task/internal/config"
	"api-task/internal/delivery/http/handler"
	"api-task/internal/delivery/http/middleware"
	"api-task/internal/delivery/http/router"
	"api-task/internal/platform/auth"
	"api-task/internal/platform/database"
	"api-task/internal/platform/logger"
	"api-task/internal/repository/postgres"
	"api-task/internal/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.NewLogger(cfg)

	db, err := database.NewPostgresDB(cfg, log)
	if err != nil {
		log.Fatal().Err(err).Msg("connect to database")
	}

	userRepo := postgres.NewUserRepository(db)
	userUC := usecase.NewUserUsecase(userRepo, log)
	userHandler := handler.NewUserHandler(userUC)

	taskRepo := postgres.NewTaskRepository(db)
	taskUC := usecase.NewTaskUsecase(taskRepo, userRepo, log)
	taskHandler := handler.NewTaskHandler(taskUC)

	roleRepo := postgres.NewRoleRepository(db)
	tokenManager, err := auth.NewManager(cfg.Auth.Secret, cfg.Auth.TTL)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid auth configuration")
	}
	authUC := usecase.NewAuthUsecase(userRepo, roleRepo, tokenManager, log)
	roleUC := usecase.NewRoleUsecase(userRepo, roleRepo, log)
	authHandler := handler.NewAuthHandler(authUC)
	roleHandler := handler.NewRoleHandler(roleUC)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.NewErrorHandler(log),
	})

	app.Use(middleware.Recover(log))
	app.Use(requestid.New())
	app.Use(middleware.NewRequestLogger(log))

	app.Hooks().OnPreShutdown(func() error {
		log.Info().Msg("shutting down, draining in-flight requests")
		return nil
	})
	app.Hooks().OnPostShutdown(func(shutdownErr error) error {
		log.Info().Msg("server stopped, closing database pool")
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("close database pool")
		}
		return shutdownErr
	})

	router.RegisterRoutes(app, router.Handlers{
		Users: userHandler,
		Tasks: taskHandler,
		Auth:  authHandler,
		Roles: roleHandler,
	}, tokenManager)

	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Listen(":"+cfg.App.Port, fiber.ListenConfig{
		GracefulContext:       shutdownCtx,
		ShutdownTimeout:       10 * time.Second,
		DisableStartupMessage: cfg.App.Env == "production",
	}); err != nil && shutdownCtx.Err() == nil {
		log.Fatal().Err(err).Msg("listen failed")
	}

	log.Info().Msg("application exited")
}
