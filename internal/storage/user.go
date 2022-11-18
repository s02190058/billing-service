package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s02190058/billing-service/internal/model"
	"github.com/s02190058/billing-service/internal/service"
	"go.uber.org/zap"
)

type UserStorage struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
}

func NewUserStorage(logger *zap.SugaredLogger, db *pgxpool.Pool) UserStorage {
	return UserStorage{
		logger: logger,
		db:     db,
	}
}

func (s UserStorage) GetBalance(id int) (int, error) {
	query := "SELECT balance FROM users WHERE id=$1"
	var balance int
	if err := s.db.QueryRow(
		context.Background(),
		query,
		id,
	).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: %d", service.ErrUserNotFound, id)
		}

		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	return balance, nil
}

func (s UserStorage) TopUpBalance(id int, amount int) (int, error) {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		s.logger.Errorf("can't begin transaction: %v", err)
		return 0, service.ErrInternalServerError
	}
	defer func() {
		if err = tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Errorf("can't rollback transcation: %v", err)
		}
	}()

	query := "UPDATE users SET balance=balance+$1 WHERE id=$2 RETURNING balance"
	var balance int
	if err = tx.QueryRow(
		context.Background(),
		query,
		amount,
		id,
	).Scan(&balance); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			s.logger.Errorf("can't process query %q: %v", query, err)
			return 0, service.ErrInternalServerError
		}

		query = "INSERT INTO users (id, balance) VALUES ($1, $2) RETURNING balance"
		if err = tx.QueryRow(
			context.Background(),
			query,
			id,
			amount,
		).Scan(&balance); err != nil {
			s.logger.Errorf("can't process query %q: %v", query, err)
			return 0, service.ErrInternalServerError
		}
	}

	query = "INSERT INTO journal (user_id, amount, message) VALUES ($1, $2, $3)"
	message := "account replenishment"
	if _, err = tx.Exec(
		context.Background(),
		query,
		id,
		amount,
		message,
	); err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	if err = tx.Commit(context.Background()); err != nil {
		s.logger.Errorf("can't commit transaction: %v", err)
		return 0, service.ErrInternalServerError
	}

	return balance, nil
}

func (s UserStorage) Transfer(id, receiverID int, amount int) (int, error) {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		s.logger.Errorf("can't begin transaction: %v", err)
		return 0, service.ErrInternalServerError
	}
	defer func() {
		if err = tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Errorf("can't rollback transcation: %v", err)
		}
	}()

	query := "UPDATE users SET balance=balance-$1 WHERE id=$2 RETURNING balance"
	var balance int
	if err = tx.QueryRow(
		context.Background(),
		query,
		amount,
		id,
	).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("%w: %d", service.ErrUserNotFound, id)
		}

		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	if balance < 0 {
		return 0, fmt.Errorf("%w: %d", service.ErrInsufficientFunds, balance+amount)
	}

	query = "UPDATE users SET balance=balance+$1 WHERE id=$2"
	tag, err := tx.Exec(
		context.Background(),
		query,
		amount,
		receiverID,
	)
	if err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	if tag.RowsAffected() == 0 {
		return 0, fmt.Errorf("%w: %d", service.ErrUserNotFound, receiverID)
	}

	query = "INSERT INTO journal (user_id, amount, message) VALUES ($1, $2, $3)"
	message := fmt.Sprintf("transfer to the user %d", receiverID)
	if _, err = tx.Exec(
		context.Background(),
		query,
		id,
		-amount,
		message,
	); err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	message = fmt.Sprintf("transfer from the user %d", receiverID)
	if _, err = tx.Exec(
		context.Background(),
		query,
		receiverID,
		amount,
		message,
	); err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return 0, service.ErrInternalServerError
	}

	if err = tx.Commit(context.Background()); err != nil {
		s.logger.Errorf("can't commit transaction: %v", err)
		return 0, service.ErrInternalServerError
	}

	return balance, nil
}

func (s UserStorage) Transactions(id int, orderField string, limit, offset int) ([]model.Transaction, error) {
	query := "SELECT id, user_id, amount, message, created " +
		"FROM journal " +
		"WHERE user_id=$1 " +
		"ORDER BY " + orderField + " " +
		"LIMIT $2 " +
		"OFFSET $3"

	rows, err := s.db.Query(
		context.Background(),
		query,
		id,
		limit,
		offset,
	)
	if err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return nil, service.ErrInternalServerError
	}

	transactions := make([]model.Transaction, 0)
	for rows.Next() {
		var transaction model.Transaction
		if err = rows.Scan(
			&transaction.ID,
			&transaction.UserID,
			&transaction.Amount,
			&transaction.Message,
			&transaction.Created,
		); err != nil {
			s.logger.Errorf("can't scan transaction values %q: %v", query, err)
			return nil, service.ErrInternalServerError
		}

		transactions = append(transactions, transaction)
	}
	if err = rows.Err(); err != nil {
		s.logger.Errorf("error occurred during rows scanning: %v", err)
		return nil, service.ErrInternalServerError
	}

	return transactions, nil
}
