package transport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/s02190058/billing-service/internal/service"
	"go.uber.org/zap"
)

var (
	ErrBadRequest    = errors.New("bad request")
	ErrMissedUserID  = errors.New("missed user id")
	ErrInvalidUserID = errors.New("user id must be an integer")
)

type userService interface {
	GetBalance(id int) (balance int, err error)
	TopUpBalance(id int, amount int) (balance int, err error)
	Transfer(id, receiverID int, amount int) (balance int, err error)
}

type userHandler struct {
	logger  *zap.SugaredLogger
	service userService
}

func registerUserRoutes(
	logger *zap.SugaredLogger, router *mux.Router, service userService) {
	handler := userHandler{
		logger:  logger,
		service: service,
	}

	router.Handle("/{user_id}", handler.handleGetBalance()).Methods(http.MethodGet)
	router.Handle("/{user_id}", handler.handleTopUpBalance()).Methods(http.MethodPost)
	router.Handle("/{user_id}/transfer", handler.handleTransfer()).Methods(http.MethodPost)
}

func getUserID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idString, ok := vars["user_id"]
	if !ok {
		return 0, ErrMissedUserID
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, ErrInvalidUserID
	}

	return id, nil
}

func balanceResponse(logger *zap.SugaredLogger, w http.ResponseWriter, code int, balance int) {
	response(logger, w, code, map[string]int{
		"balance": balance,
	})
}

func (h *userHandler) handleGetBalance() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getUserID(r)
		if err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, err)
			return
		}

		balance, err := h.service.GetBalance(id)
		if err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrUserNotFound):
				code = http.StatusNotFound
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		balanceResponse(h.logger, w, http.StatusOK, balance)
	})
}

func (h *userHandler) handleTopUpBalance() http.Handler {
	type input struct {
		Amount int `json:"amount"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getUserID(r)
		if err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, err)
			return
		}

		data := new(input)
		if err = json.NewDecoder(r.Body).Decode(data); err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrBadRequest)
			return
		}

		if err = r.Body.Close(); err != nil {
			h.logger.Errorf("can't close request body: %v", err)
		}

		balance, err := h.service.TopUpBalance(id, data.Amount)
		if err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidAmount):
				code = http.StatusBadRequest
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		balanceResponse(h.logger, w, http.StatusOK, balance)
	})
}

func (h *userHandler) handleTransfer() http.Handler {
	type input struct {
		ReceiverID int `json:"receiver_id"`
		Amount     int `json:"amount"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getUserID(r)
		if err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, err)
			return
		}

		data := new(input)
		if err = json.NewDecoder(r.Body).Decode(data); err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrBadRequest)
			return
		}

		if err = r.Body.Close(); err != nil {
			h.logger.Errorf("can't close request body: %v", err)
		}

		balance, err := h.service.Transfer(id, data.ReceiverID, data.Amount)
		if err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidAmount):
				code = http.StatusBadRequest
			case errors.Is(err, service.ErrUserNotFound):
				code = http.StatusNotFound
			case errors.Is(err, service.ErrInvalidTransfer),
				errors.Is(err, service.ErrInsufficientFunds):
				code = http.StatusUnprocessableEntity
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		balanceResponse(h.logger, w, http.StatusOK, balance)
	})
}
