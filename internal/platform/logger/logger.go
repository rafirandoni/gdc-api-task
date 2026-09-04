package logger

import (
	"os"

	"github.com/rs/zerolog"

	"api-task/internal/config"
)

func NewLogger(cfg *config.Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.App.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	log := zerolog.New(os.Stdout).With().Timestamp().Logger().Level(level)

	if err != nil {
		log.Warn().Str("configured", cfg.App.LogLevel).Msg("invalid LOG_LEVEL, defaulting to info")
	}
	return log
}
