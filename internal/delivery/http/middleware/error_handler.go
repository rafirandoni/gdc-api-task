package middleware

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	"api-task/internal/delivery/http/response"
	"api-task/internal/domain"
)

func NewErrorHandler(log zerolog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, message := classify(err)

		if status >= 500 {
			log.Error().
				Err(err).
				Str("request_id", requestID(c)).
				Msg("unhandled error")
		}

		return response.Failure(c, status, message)
	}
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"

	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden, "forbidden"

	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound, "not found"

	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict, "already exists"

	case errors.Is(err, domain.ErrInvalidInput):
		return http.StatusBadRequest, "invalid input"
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) && fiberErr.Code >= http.StatusBadRequest && fiberErr.Code < http.StatusInternalServerError {
		return fiberErr.Code, http.StatusText(fiberErr.Code)
	}

	return http.StatusInternalServerError, "internal server error"
}
