package router

import (
	"github.com/gofiber/fiber/v3"

	"api-task/internal/delivery/http/handler"
	"api-task/internal/delivery/http/middleware"
	"api-task/internal/domain"
)

type Handlers struct {
	Users *handler.UserHandler
	Tasks *handler.TaskHandler
	Auth  *handler.AuthHandler
	Roles *handler.RoleHandler
}

func RegisterRoutes(app *fiber.App, hs Handlers, tm domain.TokenManager) {
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	api := app.Group("/api/v1")

	api.Post("/auth/login", hs.Auth.Login)
	api.Post("/users", hs.Users.Register)

	usersGroup := api.Group("/users", middleware.Auth(tm))
	usersGroup.Get("/", hs.Users.List)
	usersGroup.Get("/me", hs.Users.GetMe)
	usersGroup.Get("/:id<int>", hs.Users.GetProfile)
	usersGroup.Patch("/:id<int>", hs.Users.UpdateProfile)
	usersGroup.Delete("/:id<int>", hs.Users.DeleteAccount)
	usersGroup.Post("/:id<int>/roles", hs.Roles.Grant)
	usersGroup.Delete("/:id<int>/roles/:label", hs.Roles.Revoke)

	rolesGroup := api.Group("/roles", middleware.Auth(tm))
	rolesGroup.Get("/", hs.Roles.List)

	tasksGroup := api.Group("/tasks", middleware.Auth(tm))
	tasksGroup.Post("/", hs.Tasks.Create)
	tasksGroup.Get("/", hs.Tasks.List)
	tasksGroup.Get("/:id<guid>", hs.Tasks.Get)
	tasksGroup.Patch("/:id<guid>", hs.Tasks.Update)
	tasksGroup.Post("/:id<guid>/assign", hs.Tasks.Assign)
	tasksGroup.Delete("/:id<guid>", hs.Tasks.Delete)
}
