package transport

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func ConfigureRouter(
	logger *zap.SugaredLogger,
	userService userService,
	orderService orderService,
) http.Handler {
	router := mux.NewRouter()

	registerUserRoutes(logger, router.PathPrefix("/users").Subrouter(), userService)

	registerOrderRoutes(logger, router.PathPrefix("/orders").Subrouter(), orderService)

	router.PathPrefix("/reports/").Handler(
		http.StripPrefix("/reports", http.FileServer(http.Dir("/reports"))),
	)

	mw := middleware{
		logger: logger,
	}

	router.Use(mw.catchPanic, mw.setRequestID, mw.logRequest)

	return router
}
