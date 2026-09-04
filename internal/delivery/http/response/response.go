package response

import "github.com/gofiber/fiber/v3"

type Meta struct {
	Page        int `json:"page"`
	ItemPerPage int `json:"itemPerPage"`
	Total       int `json:"total"`
}

type resWrapper struct {
	Success bool    `json:"success"`
	Data    any     `json:"data"`
	Error   *string `json:"error"`
	Meta    *Meta   `json:"meta"`
}

func Success(c fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(resWrapper{Success: true, Data: data})
}

func SuccessPage(c fiber.Ctx, status int, data any, meta Meta) error {
	return c.Status(status).JSON(resWrapper{Success: true, Data: data, Meta: &meta})
}

func Failure(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(resWrapper{Success: false, Error: &message})
}
