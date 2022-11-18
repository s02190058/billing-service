package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrBadConfig     = errors.New("bad config")
	ErrBadConnection = errors.New("can't establish connection to the postgres server")
)

type Config struct {
	User         string
	Password     string
	Host         string
	Port         string
	Database     string
	SSLMode      string
	ConnAttempts int
	ConnTimeout  time.Duration
	MaxPoolSize  int
}

func New(logger *zap.SugaredLogger, cfg Config) (*pgxpool.Pool, error) {
	url := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	logger.Infof("database url: %s", url)

	poolCfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		logger.Error("can't parse config")
		return nil, ErrBadConfig
	}

	poolCfg.MaxConns = int32(cfg.MaxPoolSize)

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, ErrBadConfig
	}

	for cfg.ConnAttempts > 0 {
		if err = pool.Ping(context.Background()); err == nil {
			break
		}

		cfg.ConnAttempts--

		logger.Infof("trying to connect to the postgres server, attempts left: %d", cfg.ConnAttempts)

		time.Sleep(cfg.ConnTimeout)
	}

	if err != nil {
		return nil, ErrBadConnection
	}

	return pool, nil
}
