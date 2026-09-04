package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"

	"api-task/internal/config"
)

func NewPostgresDB(cfg *config.Config, log zerolog.Logger) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(PostgresDSN(cfg.DB))))

	sqldb.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.DB.MaxIdleConns)

	db := bun.NewDB(sqldb, pgdialect.New())

	if cfg.App.Env != "production" {
		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.WithWriter(queryLogWriter{log: log}),
		))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqldb.PingContext(ctx); err != nil {
		_ = sqldb.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}

func PostgresDSN(cfg config.DB) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   "/" + cfg.Name,
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

type queryLogWriter struct {
	log zerolog.Logger
}

func (w queryLogWriter) Write(p []byte) (int, error) {
	if msg := strings.TrimSpace(string(p)); msg != "" {
		w.log.Debug().Msg(msg)
	}
	return len(p), nil
}
