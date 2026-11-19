package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entitiesCart "e-commerce/core/entities/cart"
	entitiesUser "e-commerce/core/entities/user"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type cartRepository struct {
	db *sql.DB
}

func NewCartRepository(db *sql.DB) entitiesCart.CartRepository {
	return &cartRepository{
		db: db,
	}
}

func (r *cartRepository) CreateCart(cart entitiesCart.Cart) error {

	status := cart.Status
	if status == "" {
		status = entitiesCart.CartStatusActive
	}

	statement, err := r.db.Prepare("INSERT INTO carts (id, user_id, status) VALUES (?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(cart.Id, cart.User.ID, status); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrConflict
		}

		return err
	}

	return nil
}

func (r *cartRepository) ReadCartByUserId(userId string, status string) (*entitiesCart.Cart, error) {

	row, err := r.db.Query("SELECT"+
		" carts.id AS cart_id,"+
		" carts.status AS cart_status,"+
		" carts.updated_at AS cart_updated_at, "+
		" carts.created_at AS cart_created_at,"+
		" users.id AS user_id,"+
		" users.updated_at user_updated_at,"+
		" users.created_at user_created_at,"+
		" users.user_name,"+
		" users.user_role"+
		" FROM carts"+
		" INNER JOIN users ON carts.user_id = users.id"+
		" WHERE carts.user_id = ? AND carts.status = ?", userId, status)

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

	cart := entitiesCart.Cart{
		User: &entitiesUser.User{},
	}

	if err = row.Scan(
		&cart.Id,
		&cart.Status,
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
