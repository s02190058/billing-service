package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s02190058/billing-service/internal/service"
	"go.uber.org/zap"
)

type OrderStorage struct {
	logger *zap.SugaredLogger
	db     *pgxpool.Pool
}

func NewOrderStorage(logger *zap.SugaredLogger, db *pgxpool.Pool) OrderStorage {
	return OrderStorage{
		logger: logger,
		db:     db,
	}
}

func (s OrderStorage) Reserve(orderID, userID, serviceID int, cost int) error {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		s.logger.Errorf("can't begin transaction: %v", err)
		return service.ErrInternalServerError
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
		cost,
		userID,
	).Scan(&balance); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %d", service.ErrUserNotFound, userID)
		}

		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if balance < 0 {
		return fmt.Errorf("%w: %d", service.ErrInsufficientFunds, balance+cost)
	}

	query = "INSERT INTO reserves (order_id, user_id, service_id, cost, status) " +
		"VALUES ($1, $2, $3, $4, $5)"

	status := "reserved"
	if _, err = tx.Exec(
		context.Background(),
		query,
		orderID,
		userID,
		serviceID,
		cost,
		status,
	); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			// unique_violation
			case "23505":
				return fmt.Errorf("%w: (%d,%d,%d)",
					service.ErrAlreadyReserved,
					orderID,
					userID,
					serviceID,
				)
			}
		}

		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if err = tx.Commit(context.Background()); err != nil {
		s.logger.Errorf("can't commit transaction: %v", err)
		return service.ErrInternalServerError
	}

	return nil
}

func (s OrderStorage) Confirm(orderID, userID, serviceID int, cost int) error {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		s.logger.Errorf("can't begin transaction: %v", err)
		return service.ErrInternalServerError
	}
	defer func() {
		if err = tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Errorf("can't rollback transcation: %v", err)
		}
	}()

	query := "UPDATE reserves SET status=$1 WHERE " +
		"order_id=$2 AND user_id=$3 AND service_id=$4 AND cost=$5 AND status=$6"

	status := "confirmed"
	prevStatus := "reserved"
	tag, err := tx.Exec(
		context.Background(),
		query,
		status,
		orderID,
		userID,
		serviceID,
		cost,
		prevStatus,
	)
	if err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"%w: (%d,%d,%d)",
			service.ErrRecordNotFound,
			orderID,
			userID,
			serviceID,
		)
	}

	query = "INSERT INTO journal (user_id, amount, message) VALUES ($1, $2, $3)"
	message := fmt.Sprintf("payment for the service %d", serviceID)
	if _, err = tx.Exec(
		context.Background(),
		query,
		userID,
		-cost,
		message,
	); err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if err = tx.Commit(context.Background()); err != nil {
		s.logger.Errorf("can't commit transaction: %v", err)
		return service.ErrInternalServerError
	}

	return nil
}

func (s OrderStorage) Reject(orderID, userID, serviceID int, cost int) error {
	tx, err := s.db.Begin(context.Background())
	if err != nil {
		s.logger.Errorf("can't begin transaction: %v", err)
		return service.ErrInternalServerError
	}
	defer func() {
		if err = tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			s.logger.Errorf("can't rollback transcation: %v", err)
		}
	}()

	query := "UPDATE reserves SET status=$1 WHERE " +
		"order_id=$2 AND user_id=$3 AND service_id=$4 AND cost=$5 AND status=$6"

	status := "confirmed"
	prevStatus := "reserved"
	tag, err := tx.Exec(
		context.Background(),
		query,
		status,
		orderID,
		userID,
		serviceID,
		cost,
		prevStatus,
	)
	if err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf(
			"%w: (%d,%d,%d)",
			service.ErrRecordNotFound,
			orderID,
			userID,
			serviceID,
		)
	}

	query = "UPDATE users SET balance=balance+$1 WHERE id=$2"
	if _, err = tx.Exec(
		context.Background(),
		query,
		cost,
		userID,
	); err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return service.ErrInternalServerError
	}

	if err = tx.Commit(context.Background()); err != nil {
		s.logger.Errorf("can't commit transaction: %v", err)
		return service.ErrInternalServerError
	}

	return nil
}

func (s OrderStorage) Report(year int, month time.Month) ([][]string, error) {
	from := time.Date(year, month, 0, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	query := "SELECT service_id, SUM(cost) AS total_revenue FROM reserves " +
		"WHERE status=$1 AND created>=$2 AND created<$3 " +
		"GROUP BY service_id"

	status := "confirmed"
	rows, err := s.db.Query(
		context.Background(),
		query,
		status,
		from,
		to,
	)
	if err != nil {
		s.logger.Errorf("can't process query %q: %v", query, err)
		return nil, service.ErrInternalServerError
	}

	services := make([][]string, 0)
	for rows.Next() {
		var serviceID string
		var totalRevenue string
		if err = rows.Scan(&serviceID, &totalRevenue); err != nil {
			s.logger.Errorf("can't scan service values")
			return nil, service.ErrInternalServerError
		}

		services = append(services, []string{serviceID, totalRevenue})
	}

	return services, nil
}
