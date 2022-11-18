package app

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/s02190058/billing-service/internal/config"
	"github.com/s02190058/billing-service/internal/service"
	"github.com/s02190058/billing-service/internal/storage"
	"github.com/s02190058/billing-service/internal/transport"
	"github.com/s02190058/billing-service/pkg/httpserver"
	"github.com/s02190058/billing-service/pkg/postgres"
	"github.com/s02190058/billing-service/pkg/zaplogger"
)

func Run(cfg *config.Config) {
	logger := zaplogger.New(cfg.Logger.Level)

	pool, err := postgres.New(logger, postgres.Config(cfg.Postgres))
	if err != nil {
		logger.Fatal(err)
	}
	defer pool.Close()

	userStorage := storage.NewUserStorage(logger, pool)
	userService := service.NewUserService(userStorage)

	router := transport.ConfigureRouter(logger, userService)

	server := httpserver.New(router, httpserver.Config(cfg.Server))

	logger.Infof("starting http server on port %s", cfg.Server.Port)
	server.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Infof("server interrupt: %v", sig)
	case err := <-server.Notify():
		logger.Errorf("error occurred since server started: %v", err)
	}

	if err := server.Shutdown(); err != nil {
		logger.Errorf("error occurred during server shutdown: %v", err)
	}
}
