package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv    = "development"
	defaultAppPort   = "8080"
	defaultLogLevel  = "info"
	defaultDBHost    = "localhost"
	defaultDBPort    = "5432"
	defaultDBUser    = "postgres"
	defaultDBSSLMode = "disable"
	defaultMaxOpen   = 20
	defaultMaxIdle   = 10
	defaultJWTTTL    = 24 * time.Hour
)

type App struct {
	Env      string
	Port     string
	LogLevel string
}

type DB struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

type Auth struct {
	Secret string
	TTL    time.Duration
}

type Config struct {
	App  App
	DB   DB
	Auth Auth
}

func Load() (*Config, error) {
	cfg := &Config{}

	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		appEnv = defaultAppEnv
	}

	if appEnv != "production" {
		if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("load .env file: %w", err)
		}
		if fromFile := os.Getenv("APP_ENV"); fromFile != "" {
			appEnv = fromFile
		}
	}
	cfg.App.Env = appEnv

	cfg.App.Port = getEnv("APP_PORT", defaultAppPort)
	cfg.App.LogLevel = getEnv("LOG_LEVEL", defaultLogLevel)

	cfg.DB.Host = getEnv("DB_HOST", defaultDBHost)
	cfg.DB.Port = getEnv("DB_PORT", defaultDBPort)
	cfg.DB.User = getEnv("DB_USER", defaultDBUser)
	cfg.DB.SSLMode = getEnv("DB_SSLMODE", defaultDBSSLMode)

	if cfg.DB.Password = os.Getenv("DB_PASSWORD"); cfg.DB.Password == "" {
		return nil, errors.New("DB_PASSWORD is required")
	}
	if cfg.DB.Name = os.Getenv("DB_NAME"); cfg.DB.Name == "" {
		return nil, errors.New("DB_NAME is required")
	}

	maxOpen, err := getInt("DB_MAX_OPEN_CONNS", defaultMaxOpen)
	if err != nil {
		return nil, err
	}
	cfg.DB.MaxOpenConns = maxOpen

	maxIdle, err := getInt("DB_MAX_IDLE_CONNS", defaultMaxIdle)
	if err != nil {
		return nil, err
	}
	cfg.DB.MaxIdleConns = maxIdle

	if cfg.Auth.Secret = os.Getenv("JWT_SECRET"); cfg.Auth.Secret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	ttl := defaultJWTTTL
	if raw := os.Getenv("JWT_TTL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("JWT_TTL must be a duration like 24h or 1h30m, got %q", raw)
		}
		ttl = parsed
	}
	cfg.Auth.TTL = ttl

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) (int, error) {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, raw)
	}
	return v, nil
}
