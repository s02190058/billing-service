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
	ErrMissedOrderID  = errors.New("missed order id")
	ErrInvalidOrderID = errors.New("order id must be an integer")
	ErrMissedYear     = errors.New("missed year")
	ErrInvalidYear    = errors.New("year must be an integer")
	ErrMissedMonth    = errors.New("missed month")
	ErrInvalidMonth   = errors.New("month must be an integer")
)

type orderService interface {
	Reserve(orderID, userID, serviceID int, cost int) (err error)
	Confirm(orderID, userID, serviceID int, cost int) (err error)
	Reject(orderID, userID, serviceID int, cost int) (err error)
	Report(year, month int) (path string, err error)
}

type orderHandler struct {
	logger  *zap.SugaredLogger
	service orderService
}

func registerOrderRoutes(logger *zap.SugaredLogger, router *mux.Router, service orderService) {
	handler := orderHandler{
		logger:  logger,
		service: service,
	}

	router.Handle("/{order_id}/reserve", handler.HandleReserve()).Methods(http.MethodPost)
	router.Handle("/{order_id}/confirm", handler.HandleConfirm()).Methods(http.MethodPost)
	router.Handle("/{order_id}/reject", handler.HandleReject()).Methods(http.MethodPost)
	router.Handle("/report", handler.handleReport()).Methods(http.MethodGet)
}

func getOrderID(r *http.Request) (int, error) {
	vars := mux.Vars(r)
	idString, ok := vars["order_id"]
	if !ok {
		return 0, ErrMissedOrderID
	}

	id, err := strconv.Atoi(idString)
	if err != nil {
		return 0, ErrInvalidOrderID
	}

	return id, nil
}

func statusResponse(logger *zap.SugaredLogger, w http.ResponseWriter, code int, status string) {
	response(logger, w, code, map[string]string{
		"status": status,
	})
}

func (h *orderHandler) HandleReserve() http.Handler {
	type input struct {
		UserID    int `json:"user_id"`
		ServiceID int `json:"service_id"`
		Cost      int `json:"cost"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getOrderID(r)
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

		if err = h.service.Reserve(id, data.UserID, data.ServiceID, data.Cost); err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidCost):
				code = http.StatusBadRequest
			case errors.Is(err, service.ErrUserNotFound):
				code = http.StatusNotFound
			case errors.Is(err, service.ErrInsufficientFunds):
				code = http.StatusUnprocessableEntity
			case errors.Is(err, service.ErrAlreadyReserved):
				code = http.StatusBadRequest
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		statusResponse(h.logger, w, http.StatusOK, "reserved")
	})
}

func (h *orderHandler) HandleConfirm() http.Handler {
	type input struct {
		UserID    int `json:"user_id"`
		ServiceID int `json:"service_id"`
		Cost      int `json:"cost"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getOrderID(r)
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

		if err = h.service.Confirm(id, data.UserID, data.ServiceID, data.Cost); err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidCost):
				code = http.StatusBadRequest
			case errors.Is(err, service.ErrRecordNotFound):
				code = http.StatusNotFound
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		statusResponse(h.logger, w, http.StatusOK, "confirmed")
	})
}

func (h *orderHandler) HandleReject() http.Handler {
	type input struct {
		UserID    int `json:"user_id"`
		ServiceID int `json:"service_id"`
		Cost      int `json:"cost"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := getOrderID(r)
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

		if err = h.service.Reject(id, data.UserID, data.ServiceID, data.Cost); err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidCost):
				code = http.StatusBadRequest
			case errors.Is(err, service.ErrRecordNotFound):
				code = http.StatusNotFound
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		statusResponse(h.logger, w, http.StatusOK, "rejected")
	})
}

func (h *orderHandler) handleReport() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()

		if params.Get("year") == "" {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrMissedYear)
			return
		}
		year, err := strconv.Atoi(params.Get("year"))
		if err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrInvalidYear)
			return
		}

		if params.Get("month") == "" {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrMissedMonth)
			return
		}
		month, err := strconv.Atoi(params.Get("month"))
		if err != nil {
			errorResponse(h.logger, w, http.StatusBadRequest, ErrInvalidMonth)
			return
		}

		path, err := h.service.Report(year, month)
		if err != nil {
			var code int
			switch {
			case errors.Is(err, service.ErrInvalidMonth):
				code = http.StatusBadRequest
			default:
				code = http.StatusInternalServerError
			}
			errorResponse(h.logger, w, code, err)
			return
		}

		response(h.logger, w, http.StatusOK, map[string]string{
			"url": r.Host + path,
		})
	})
}
