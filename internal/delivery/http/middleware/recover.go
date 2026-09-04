package middleware

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"
)

func Recover(log zerolog.Logger) fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e any) {
			log.Error().
				Err(fmt.Errorf("%v", e)).
				Str("request_id", requestID(c)).
				Msgf("panic recovered:\n%s", debug.Stack())
		},
	})
}
