package service

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	ErrAlreadyReserved = errors.New("service has already been reserved")
	ErrInvalidCost     = errors.New("cost must be non-negative")
	ErrInvalidMonth    = errors.New("month must be in the range from 1 to 12")
	ErrRecordNotFound  = errors.New("record not found")
)

type orderStorage interface {
	Reserve(orderID, userID, serviceID int, cost int) (err error)
	Confirm(orderID, userID, serviceID int, cost int) (err error)
	Reject(orderID, userID, serviceID int, cost int) (err error)
	Report(year int, month time.Month) (services [][]string, err error)
}

type OrderService struct {
	storage orderStorage
}

func NewOrderService(storage orderStorage) OrderService {
	return OrderService{
		storage: storage,
	}
}

func (s OrderService) Reserve(orderID, userID, serviceID int, cost int) error {
	if cost < 0 {
		return ErrInvalidCost
	}

	return s.storage.Reserve(orderID, userID, serviceID, cost)
}

func (s OrderService) Confirm(orderID, userID, serviceID int, cost int) error {
	if cost < 0 {
		return ErrInvalidCost
	}

	return s.storage.Confirm(orderID, userID, serviceID, cost)
}

func (s OrderService) Reject(orderID, userID, serviceID int, cost int) error {
	if cost < 0 {
		return ErrInvalidCost
	}

	return s.storage.Reject(orderID, userID, serviceID, cost)
}

func (s OrderService) Report(year, month int) (string, error) {
	if month < 1 || month > 12 {
		return "", ErrInvalidMonth
	}

	services, err := s.storage.Report(year, time.Month(month))
	if err != nil {
		return "", err
	}

	reportPath := fmt.Sprintf("/reports/%d-%d.csv", year, month)
	report, err := os.Create(reportPath)
	if err != nil {
		println(err.Error())
		return "", ErrInternalServerError
	}

	csvWriter := csv.NewWriter(report)
	csvWriter.Comma = ';'

	if err = csvWriter.Write([]string{"service", "total revenue"}); err != nil {
		return "", ErrInternalServerError
	}
	if err = csvWriter.WriteAll(services); err != nil {
		return "", ErrInternalServerError
	}

	return reportPath, nil
}
