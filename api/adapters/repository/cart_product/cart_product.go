package repository

import (
	"database/sql"
	"e-commerce/core/apperrors"
	entities "e-commerce/core/entities/cart_product"
	"errors"

	"github.com/go-sql-driver/mysql"
)

type cartProductRepository struct {
	db *sql.DB
}

func NewCartProductRepository(db *sql.DB) entities.CartProductRepository {
	return &cartProductRepository{
		db: db,
	}
}

func (r *cartProductRepository) AddProductCart(cartProduct entities.CartProduct) error {

	statement, err := r.db.Prepare("INSERT INTO cart_products " +
		"(id, cart_id, product_id,quantity ,isSelect)" +
		"VALUES (?,?,?,?)")

	if err != nil {
		return err
	}

	defer statement.Close()

	if _, err = statement.Exec(
		cartProduct.Id,
		cartProduct.Cart.Id,
		cartProduct.Product.Id,
		cartProduct.Quantity,
		cartProduct.IsSelect,
	); err != nil {

		var sqlErr *mysql.MySQLError

		if errors.As(err, &sqlErr) && sqlErr.Number == 1062 {
			return apperrors.ErrNotFound
		}

		return err
	}

	return nil
}

func (r *cartProductRepository) ReadAllProductsCart(userId string) ([]entities.CartProduct, error) {

	rows, err := r.db.Query("SELECT cartproducts.* FROM cart_products cartproducts "+
		"JOIN carts cart ON cartproducts.cart_id = cart.id"+
		"JOIN users user ON cart.user_id = user.id"+
		"WHERE user.id = ?", userId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var listProductCarts []entities.CartProduct

	for rows.Next() {

		var cartProduct entities.CartProduct

		if err = rows.Scan(
			cartProduct.Id,
			cartProduct.Cart.Id,
			cartProduct.Product.Id,
			cartProduct.Quantity,
			cartProduct.IsSelect,
			cartProduct.UpdateAt,
			cartProduct.CreateAt,
		); err != nil {

			return nil, err
		}

		listProductCarts = append(listProductCarts, cartProduct)

	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return listProductCarts, nil
}

func (r *cartProductRepository) RemoveProductCart(id string) error {
	return nil
}

func (r *cartProductRepository) UpdateIsSelect(bool, cartProductId string) error {
	return nil
}
