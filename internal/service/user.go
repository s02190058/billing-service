package service

import "errors"

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrInvalidAmount     = errors.New("amount must be positive")
	ErrInvalidTransfer   = errors.New("impossible to transfer to yourself")
	ErrUserNotFound      = errors.New("user not found")
)

type userStorage interface {
	GetBalance(id int) (balance int, err error)
	TopUpBalance(id int, amount int) (balance int, err error)
	Transfer(id, receiverID int, amount int) (balance int, err error)
}

type UserService struct {
	storage userStorage
}

func NewUserService(storage userStorage) UserService {
	return UserService{
		storage: storage,
	}
}

func (s UserService) GetBalance(id int) (int, error) {
	return s.storage.GetBalance(id)
}

func (s UserService) TopUpBalance(id int, amount int) (int, error) {
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}

	return s.storage.TopUpBalance(id, amount)
}

func (s UserService) Transfer(id, receiverID int, amount int) (int, error) {
	if id == receiverID {
		return 0, ErrInvalidTransfer
	}
	if amount <= 0 {
		return 0, ErrInvalidAmount
	}

	return s.storage.Transfer(id, receiverID, amount)
}
