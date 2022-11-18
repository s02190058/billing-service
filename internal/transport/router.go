package transport

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func ConfigureRouter(
	logger *zap.SugaredLogger,
	userService userService,
) http.Handler {
	router := mux.NewRouter()

	registerUserRoutes(logger, router.PathPrefix("/users").Subrouter(), userService)

	return router
}
