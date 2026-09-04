package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"
)

func NewRequestLogger(log zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		chainErr := c.Next()
		if chainErr != nil {
			if handlerErr := c.App().ErrorHandler(c, chainErr); handlerErr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		status := c.Response().StatusCode()

		logger := log.Info()
		switch {
		case status >= 500:
			logger = log.Error()
		case status >= 400:
			logger = log.Warn()
		}

		logger.
			Str("request_id", requestID(c)).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", status).
			Dur("latency", time.Since(start)).
			Msg("http request")

		return nil
	}
}

func requestID(c fiber.Ctx) string {
	return requestid.FromContext(c)
}
