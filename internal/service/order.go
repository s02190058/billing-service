package service

import "errors"

var (
	ErrAlreadyReserved = errors.New("service has already been reserved")
	ErrInvalidCost     = errors.New("cost must be non-negative")
	ErrRecordNotFound  = errors.New("record not found")
)

type orderStorage interface {
	Reserve(orderID, userID, serviceID int, cost int) (err error)
	Confirm(orderID, userID, serviceID int, cost int) (err error)
	Reject(orderID, userID, serviceID int, cost int) (err error)
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
