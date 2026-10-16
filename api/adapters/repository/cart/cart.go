package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type cartRepository struct {
	db *sql.DB
}

func NewCartRepository(db *sql.DB) entities.CartRepository {
	return &cartRepository{
		db: db,
	}
}

func (r *cartRepository) CreateCart(cart entities.Cart) error {

	statement, err := r.db.Prepare("INSERT INTO carts (id, user_id) VALUES (?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(cart.Id, cart.User.ID); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	return nil
}

func (r *cartRepository) ReadCartByUserId(userId string) (*entities.Cart, error) {

	row, err := r.db.Query("SELECT"+
		"carts_id AS cart_id"+
		"carts_updated_at AS cart_updated_at "+
		"carts_created_at AS cart_created_at"+
		"users_id AS user_id"+
		"users_updated_at user_updated_at"+
		"users_created_at user_created_at"+
		"users.user_name"+
		"users.user_role"+
		"SELECT FROM carts"+
		"INNER JOIN users ON carts.user_id = users_id"+
		"WHERE carts_user_id = ?", userId)

	if err != nil {
		return nil, err
	}

	defer row.Close()

	if !row.Next() {

		if err = row.Err(); err != nil {
			return nil, err
		}

		return nil, apperrors.ErrNotFound
	}

	var cart entities.Cart

	if err = row.Scan(
		&cart.Id,
		&cart.UpdateAt,
		&cart.CreateAt,
		&cart.User.ID,
		&cart.User.UpdatedAt,
		&cart.User.CreatedAt,
		&cart.User.Name,
		&cart.User.Role,
	); err != nil {
		return nil, err
	}

	return &cart, nil
}
