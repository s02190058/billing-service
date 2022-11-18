package transport

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type middleware struct {
	logger *zap.SugaredLogger
}

type ctxRequestIDKey int

var requestIDKey ctxRequestIDKey

func (m *middleware) catchPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.logger.Errorf("panic occured: %v", err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (m *middleware) setRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

func (m *middleware) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := m.logger.With(
			"remote_addr", r.RemoteAddr,
			"request_id", r.Context().Value(requestIDKey),
		)

		logger.Infof("started %s %s", r.Method, r.RequestURI)

		start := time.Now()
		rw := &responseWriter{
			ResponseWriter: w,
			code:           http.StatusOK,
		}
		next.ServeHTTP(rw, r)

		logger.Infof(
			"completed with %d %s in %v",
			rw.code,
			http.StatusText(rw.code),
			time.Since(start),
		)
	})
}
