package repository

import (
	"database/sql"
	entities "e-commerce/core/entities/order_status_history"
)

type orderStatusHistoryRepository struct {
	db *sql.DB
}

func NewOrderStatusHistoryRepository(db *sql.DB) entities.OrderStatusHistoryRepository {
	return &orderStatusHistoryRepository{
		db: db,
	}
}

func (r *orderStatusHistoryRepository) ReadHistoryByOrderId(orderId string) ([]entities.OrderStatusHistory, error) {

	rows, err := r.db.Query("SELECT id, order_id, status, created_at FROM order_status_history"+
		" WHERE order_id = ? ORDER BY created_at ASC", orderId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listHistory []entities.OrderStatusHistory

	for rows.Next() {

		var history entities.OrderStatusHistory

		if err = rows.Scan(
			&history.Id,
			&history.OrderId,
			&history.Status,
			&history.CreateAt,
		); err != nil {
			return nil, err
		}

		listHistory = append(listHistory, history)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listHistory, nil
}
