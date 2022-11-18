package transport

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func errorResponse(logger *zap.SugaredLogger, w http.ResponseWriter, code int, err error) {
	response(logger, w, code, map[string]string{
		"message": err.Error(),
	})
}

func response(logger *zap.SugaredLogger, w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Errorf("can't write the server response: %v", err)
	}
}
